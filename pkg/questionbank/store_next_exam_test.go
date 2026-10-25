package questionbank

import (
	"errors"
	"strings"
	"testing"
)

func completedNextExamStore(t *testing.T) *Store {
	t.Helper()
	s := automaticBank(t)
	add(t, s, Completed)
	if _, dirty := s.Snapshot(); dirty {
		t.Fatal("测试前置分段不应为 dirty")
	}
	return s
}

func TestSealCurrentForNextSealsActiveAndNextBeginCreatesDifferentSession(t *testing.T) {
	s := completedNextExamStore(t)
	before, _ := s.Snapshot()

	result, err := s.SealCurrentForNext()
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != before.ID || result.PreviousState != Active || result.SessionState != Ended || !result.Sealed {
		t.Fatalf("封存结果错误: %+v", result)
	}
	after, dirty := s.Snapshot()
	if dirty || after.ID != before.ID || after.State != Ended || after.EndedAt == nil {
		t.Fatalf("封存后的当前分段错误: session=%+v dirty=%v", after, dirty)
	}
	persisted := diskSession(t, s, before.ID)
	if persisted.State != Ended || persisted.EndedAt == nil {
		t.Fatalf("封存状态未原子落盘: %+v", persisted)
	}

	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "next-model", "next-scene", true)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := s.Snapshot()
	if recordID == "" || next.ID == before.ID || next.State != Active || len(next.Records) != 1 {
		t.Fatalf("下一次 BeginWithContext 未懒创建不同分段: before=%s next=%+v", before.ID, next)
	}
}

func TestSealCurrentForNextSealsPaused(t *testing.T) {
	s := completedNextExamStore(t)
	before, _ := s.Snapshot()
	reopened, err := NewStore(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Open(before.ID); err != nil {
		t.Fatal(err)
	}
	paused, dirty := reopened.Snapshot()
	if dirty || paused.State != Paused {
		t.Fatalf("测试前置未得到已落盘 paused 分段: %+v dirty=%v", paused, dirty)
	}

	result, err := reopened.SealCurrentForNext()
	if err != nil {
		t.Fatal(err)
	}
	if result.PreviousState != Paused || result.SessionState != Ended || !result.Sealed {
		t.Fatalf("paused 封存结果错误: %+v", result)
	}
}

func TestSealCurrentForNextEndedAndExportedAreIdempotent(t *testing.T) {
	t.Run("ended", func(t *testing.T) {
		s := completedNextExamStore(t)
		id, err := s.SealCurrent()
		if err != nil {
			t.Fatal(err)
		}
		before, _ := s.Snapshot()
		result, err := s.SealCurrentForNext()
		if err != nil {
			t.Fatal(err)
		}
		after, dirty := s.Snapshot()
		if result.SessionID != id || result.PreviousState != Ended || result.SessionState != Ended || result.Sealed || dirty {
			t.Fatalf("ended 幂等结果错误: result=%+v session=%+v dirty=%v", result, after, dirty)
		}
		if after.EndedAt == nil || !after.EndedAt.Equal(*before.EndedAt) {
			t.Fatal("ended 幂等调用改写了 EndedAt")
		}
	})

	t.Run("exported", func(t *testing.T) {
		s := completedNextExamStore(t)
		id, err := s.SealCurrent()
		if err != nil {
			t.Fatal(err)
		}
		exported, err := s.ExportSession(id, ExportOptions{Name: "下一套前导出"})
		if err != nil {
			t.Fatal(err)
		}
		before, _ := s.Snapshot()
		result, err := s.SealCurrentForNext()
		if err != nil {
			t.Fatal(err)
		}
		after, dirty := s.Snapshot()
		if result.PreviousState != Exported || result.SessionState != Exported || result.Sealed || dirty {
			t.Fatalf("exported 幂等结果错误: result=%+v session=%+v dirty=%v", result, after, dirty)
		}
		if after.ExportedAt == nil || before.ExportedAt == nil || !after.ExportedAt.Equal(*before.ExportedAt) || after.ExportDirectory != exported.Directory || len(after.Exports) != len(before.Exports) {
			t.Fatalf("exported 幂等调用破坏导出元数据: before=%+v after=%+v", before, after)
		}
	})
}

func TestSealCurrentForNextRejectsUnsafeStates(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T) *Store
		want    string
	}{
		{
			name: "nil current",
			prepare: func(t *testing.T) *Store {
				return automaticBank(t)
			},
			want: "暂无记录",
		},
		{
			name: "empty current",
			prepare: func(t *testing.T) *Store {
				s := automaticBank(t)
				if err := s.Start("单位", "综合"); err != nil {
					t.Fatal(err)
				}
				return s
			},
			want: "暂无记录",
		},
		{
			name: "discarded",
			prepare: func(t *testing.T) *Store {
				s := completedNextExamStore(t)
				if err := s.DiscardCurrent(); err != nil {
					t.Fatal(err)
				}
				return s
			},
			want: "已清空",
		},
		{
			name: "running",
			prepare: func(t *testing.T) *Store {
				s := automaticBank(t)
				if _, err := s.Begin([]string{shot(t, false)}, "model", "scene"); err != nil {
					t.Fatal(err)
				}
				return s
			},
			want: "正在处理",
		},
		{
			name: "dirty",
			prepare: func(t *testing.T) *Store {
				s := completedNextExamStore(t)
				s.dirty = true
				return s
			},
			want: "未落盘",
		},
		{
			name: "pending completion",
			prepare: func(t *testing.T) *Store {
				s := completedNextExamStore(t)
				current, _ := s.Snapshot()
				s.pendingCompletion = cloneSession(current)
				return s
			},
			want: "等待重试保存",
		},
		{
			name: "write blocked",
			prepare: func(t *testing.T) *Store {
				s := completedNextExamStore(t)
				s.writeBlocked = errors.New("synthetic write block")
				return s
			},
			want: "写入已阻止",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.prepare(t)
			before, dirtyBefore := s.Snapshot()
			result, err := s.SealCurrentForNext()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("未按预期拒绝: result=%+v err=%v", result, err)
			}
			after, dirtyAfter := s.Snapshot()
			if dirtyAfter != dirtyBefore {
				t.Fatalf("拒绝后 dirty 变化: before=%v after=%v", dirtyBefore, dirtyAfter)
			}
			if before == nil {
				if after != nil {
					t.Fatalf("拒绝后预建了分段: %+v", after)
				}
				return
			}
			if after == nil || after.ID != before.ID || after.State != before.State {
				t.Fatalf("拒绝后当前分段变化: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestSealCurrentForNextSaveFailureRestoresSessionAndDirty(t *testing.T) {
	s := completedNextExamStore(t)
	before, dirtyBefore := s.Snapshot()
	s.saveSessionHook = func(session *Session, _ string) error {
		if session.ID == before.ID && session.State == Ended {
			return errors.New("synthetic seal failure")
		}
		return nil
	}

	result, err := s.SealCurrentForNext()
	if err == nil || !strings.Contains(err.Error(), "synthetic seal failure") {
		t.Fatalf("保存失败未返回: result=%+v err=%v", result, err)
	}
	after, dirtyAfter := s.Snapshot()
	if dirtyAfter != dirtyBefore || after.ID != before.ID || after.State != Active || after.EndedAt != nil {
		t.Fatalf("保存失败后内存未恢复: before=%+v after=%+v dirtyBefore=%v dirtyAfter=%v", before, after, dirtyBefore, dirtyAfter)
	}
	persisted := diskSession(t, s, before.ID)
	if persisted.State != Active || persisted.EndedAt != nil {
		t.Fatalf("保存失败后磁盘状态变化: %+v", persisted)
	}
}
