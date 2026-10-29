package questionbank

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func completedDeleteStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	s := automaticBank(t)
	recordID, err := s.Begin([]string{shot(t, false), shot(t, true)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(recordID, "测试答案", Completed); err != nil {
		t.Fatal(err)
	}
	current, dirty := s.Snapshot()
	if current == nil || dirty {
		t.Fatalf("测试前置分段无效: current=%+v dirty=%v", current, dirty)
	}
	return s, current.ID, s.sessionDir(current.ID)
}

func TestDeleteCurrentPhysicallyRemovesSessionAndNextBeginStartsFresh(t *testing.T) {
	s, sessionID, dir := completedDeleteStore(t)
	if _, err := os.Stat(filepath.Join(dir, "session.json")); err != nil {
		t.Fatal(err)
	}
	assets, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil || len(assets) != 2 {
		t.Fatalf("测试前置截图资产错误: count=%d err=%v", len(assets), err)
	}

	deletedID, err := s.DeleteCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if deletedID != sessionID {
		t.Fatalf("删除结果 ID 错误: want=%s got=%s", sessionID, deletedID)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("当前分段目录仍存在或状态未知: %v", err)
	}
	if current, dirty := s.Snapshot(); current != nil || dirty {
		t.Fatalf("删除成功后 current 未清空: current=%+v dirty=%v", current, dirty)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, session := range list {
		if session.ID == sessionID {
			t.Fatal("物理删除分段仍出现在历史列表")
		}
	}

	recordID, err := s.Begin([]string{shot(t, false)}, "next-model", "next-scene")
	if err != nil {
		t.Fatal(err)
	}
	next, _ := s.Snapshot()
	if recordID == "" || next == nil || next.ID == sessionID || len(next.Records) != 1 {
		t.Fatalf("删除后下一题未建立新分段: %+v", next)
	}
}

func TestDeleteCurrentRejectsUnsafeStatesWithoutDeletingFiles(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T) (*Store, string)
		want    string
	}{
		{
			name: "running",
			prepare: func(t *testing.T) (*Store, string) {
				s := automaticBank(t)
				if _, err := s.Begin([]string{shot(t, false)}, "model", "scene"); err != nil {
					t.Fatal(err)
				}
				current, _ := s.Snapshot()
				return s, s.sessionDir(current.ID)
			},
			want: "正在处理",
		},
		{
			name: "dirty",
			prepare: func(t *testing.T) (*Store, string) {
				s, _, dir := completedDeleteStore(t)
				s.dirty = true
				return s, dir
			},
			want: "未落盘",
		},
		{
			name: "pending completion",
			prepare: func(t *testing.T) (*Store, string) {
				s, _, dir := completedDeleteStore(t)
				current, _ := s.Snapshot()
				s.pendingCompletion = cloneSession(current)
				return s, dir
			},
			want: "等待重试保存",
		},
		{
			name: "exported",
			prepare: func(t *testing.T) (*Store, string) {
				s, id, dir := completedDeleteStore(t)
				if _, err := s.SealCurrent(); err != nil {
					t.Fatal(err)
				}
				if _, err := s.ExportSession(id, ExportOptions{RawOnly: true}); err != nil {
					t.Fatal(err)
				}
				return s, dir
			},
			want: "已导出",
		},
		{
			name: "legacy discarded",
			prepare: func(t *testing.T) (*Store, string) {
				s, _, dir := completedDeleteStore(t)
				if err := s.DiscardCurrent(); err != nil {
					t.Fatal(err)
				}
				return s, dir
			},
			want: "已清空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, dir := tt.prepare(t)
			before, dirtyBefore := s.Snapshot()
			deletedID, err := s.DeleteCurrent()
			if err == nil || !strings.Contains(err.Error(), tt.want) || deletedID != "" {
				t.Fatalf("未按预期拒绝: id=%q err=%v", deletedID, err)
			}
			after, dirtyAfter := s.Snapshot()
			if before == nil || after == nil || after.ID != before.ID || after.State != before.State || dirtyAfter != dirtyBefore {
				t.Fatalf("拒绝后内存变化: before=%+v after=%+v", before, after)
			}
			if _, statErr := os.Stat(dir); statErr != nil {
				t.Fatalf("拒绝后分段目录被删除: %v", statErr)
			}
		})
	}
}

func TestDeleteCurrentAllowsEndedAndLegacyPausedSessions(t *testing.T) {
	t.Run("ended", func(t *testing.T) {
		s, sessionID, dir := completedDeleteStore(t)
		if _, err := s.SealCurrent(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DeleteCurrent(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("ended 分段目录仍存在: %v", err)
		}
		if current, _ := s.Snapshot(); current != nil {
			t.Fatalf("ended 删除后 current 仍存在: %+v", current)
		}
		list, err := s.List()
		if err != nil {
			t.Fatal(err)
		}
		for _, session := range list {
			if session.ID == sessionID {
				t.Fatal("ended 删除后仍在历史列表")
			}
		}
	})

	t.Run("legacy paused", func(t *testing.T) {
		legacy := automaticBank(t)
		add(t, legacy, Completed)
		legacySession, _ := legacy.Snapshot()
		legacyDir := legacy.sessionDir(legacySession.ID)

		archive := t.TempDir()
		migrated, err := NewStoreWithLegacy(archive, []string{legacy.sessionsDir()})
		if err != nil {
			t.Fatal(err)
		}
		if err = migrated.Open(legacySession.ID); err != nil {
			t.Fatal(err)
		}
		current, dirty := migrated.Snapshot()
		if current == nil || current.State != Paused || dirty {
			t.Fatalf("legacy 分段未以 paused 打开: current=%+v dirty=%v", current, dirty)
		}
		if _, err = migrated.DeleteCurrent(); err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(legacyDir); !os.IsNotExist(err) {
			t.Fatalf("legacy 当前分段目录仍存在: %v", err)
		}
		if entries, err := os.ReadDir(filepath.Join(archive, "_记录")); err == nil && len(entries) != 0 {
			t.Fatalf("删除 legacy 分段时误写主题库: %+v", entries)
		}
	})
}

func TestDeleteCurrentPartialRemoveFailureBlocksFurtherWrites(t *testing.T) {
	s, sessionID, dir := completedDeleteStore(t)
	current, _ := s.Snapshot()
	missingImage := current.Records[0].Images[0]
	s.removeAllHook = func(staging string) error {
		if err := os.Remove(filepath.Join(staging, filepath.FromSlash(missingImage))); err != nil {
			t.Fatal(err)
		}
		return errors.New("synthetic partial remove failure")
	}

	deletedID, err := s.DeleteCurrent()
	if err == nil || !strings.Contains(err.Error(), "无法完整恢复") || deletedID != "" {
		t.Fatalf("部分删除失败未进入保护状态: id=%q err=%v", deletedID, err)
	}
	current, dirty := s.Snapshot()
	if current == nil || current.ID != sessionID || dirty {
		t.Fatalf("部分删除失败后 current 标识变化: current=%+v dirty=%v", current, dirty)
	}
	if _, err = os.Stat(filepath.Join(dir, "session.json")); err != nil {
		t.Fatalf("部分删除失败后 manifest 未恢复: %v", err)
	}
	if _, err = os.Stat(filepath.Join(dir, filepath.FromSlash(missingImage))); !os.IsNotExist(err) {
		t.Fatalf("测试前置未形成资产缺失: %v", err)
	}
	if err = s.Flush(); err == nil || !strings.Contains(err.Error(), "写入已阻止") {
		t.Fatalf("资产不完整后仍允许写入: %v", err)
	}
}

func TestDeleteCurrentRemoveFailureRestoresDirectoryAndCurrent(t *testing.T) {
	s, sessionID, dir := completedDeleteStore(t)
	s.removeAllHook = func(string) error { return errors.New("synthetic remove failure") }

	deletedID, err := s.DeleteCurrent()
	if err == nil || !strings.Contains(err.Error(), "synthetic remove failure") || deletedID != "" {
		t.Fatalf("删除失败未返回: id=%q err=%v", deletedID, err)
	}
	current, dirty := s.Snapshot()
	if current == nil || current.ID != sessionID || dirty {
		t.Fatalf("删除失败后 current 变化: current=%+v dirty=%v", current, dirty)
	}
	if _, err = os.Stat(filepath.Join(dir, "session.json")); err != nil {
		t.Fatalf("删除失败后原目录未恢复: %v", err)
	}
	staging, err := filepath.Glob(filepath.Join(filepath.Dir(dir), ".deleting-"+sessionID+"-*"))
	if err != nil || len(staging) != 0 {
		t.Fatalf("删除失败后残留暂存目录: %+v err=%v", staging, err)
	}
}
