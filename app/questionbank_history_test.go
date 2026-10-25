package app

import (
	"encoding/json"
	"strings"
	"testing"

	"Q-Solver/pkg/questionbank"
)

func createQuestionBankSegments(t *testing.T, store *questionbank.Store, imageURL string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		recordID, err := store.Begin([]string{imageURL}, "model", "scene")
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Finish(recordID, "answer", questionbank.Completed); err != nil {
			t.Fatal(err)
		}
		if _, err = store.SealCurrent(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListRecentQuestionBankSessionsUsesUIOnlyLimit(t *testing.T) {
	a, store, imageURL := archiveApp(t)
	empty, err := a.ListRecentQuestionBankSessions()
	if err != nil || empty.Total != 0 || empty.Hidden != 0 || empty.Limit != 20 || len(empty.Sessions) != 0 {
		t.Fatalf("空列表错误: %+v err=%v", empty, err)
	}
	createQuestionBankSegments(t, store, imageURL, 21)
	got, err := a.ListRecentQuestionBankSessions()
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 21 || got.Hidden != 1 || got.Limit != 20 || len(got.Sessions) != 20 {
		t.Fatalf("最近 20 统计错误: %+v", got)
	}
	all, err := store.List()
	if err != nil || len(all) != 21 {
		t.Fatalf("UI 截断破坏了 Store.List 全量语义: %d err=%v", len(all), err)
	}
}

func TestQuestionBankDetailIsReadOnlyAndRejectsHiddenSession(t *testing.T) {
	a, store, imageURL := archiveApp(t)
	createQuestionBankSegments(t, store, imageURL, 21)
	all, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	currentBefore, _ := store.Snapshot()
	detail, err := a.GetQuestionBankSessionDetail(all[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	currentAfter, _ := store.Snapshot()
	if currentAfter.ID != currentBefore.ID || currentAfter.State != currentBefore.State || detail.ID != all[1].ID || len(detail.Records) != 1 {
		t.Fatalf("只读详情改变了 current 或返回错误: before=%+v after=%+v detail=%+v", currentBefore, currentAfter, detail)
	}
	encoded, _ := json.Marshal(detail)
	if strings.Contains(string(encoded), store.Root()) || strings.Contains(string(encoded), "assets/") || strings.Contains(string(encoded), "session.json") {
		t.Fatalf("详情泄露磁盘路径: %s", encoded)
	}
	if _, err = a.GetQuestionBankSessionDetail(all[20].ID); err == nil {
		t.Fatal("第 21 段仍可通过详情接口访问")
	}
	if _, err = a.GetQuestionBankRecordImage(all[20].ID, "0123456789abcdef01234567", 0); err == nil {
		t.Fatal("第 21 段仍可通过图片接口访问")
	}
}

func TestQuestionBankDiscardedDetailAndImageRemainReadable(t *testing.T) {
	a, store, imageURL := archiveApp(t)
	recordID, err := store.Begin([]string{imageURL}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Finish(recordID, "完整原答案", questionbank.Completed); err != nil {
		t.Fatal(err)
	}
	session, _ := store.Snapshot()
	if err = store.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	detail, err := a.GetQuestionBankSessionDetail(session.ID)
	if err != nil || detail.State != questionbank.Discarded || detail.Records[0].Answer != "完整原答案" {
		t.Fatalf("discarded 详情不可读: %+v err=%v", detail, err)
	}
	imageData, err := a.GetQuestionBankRecordImage(session.ID, recordID, 0)
	if err != nil || !strings.HasPrefix(imageData, "data:image/png;base64,") {
		t.Fatalf("discarded 截图不可读: %v", err)
	}
}

func TestQuestionBankDetailRejectsPayloadOverEightMiB(t *testing.T) {
	a, store, imageURL := archiveApp(t)
	recordID, err := store.Begin([]string{imageURL}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Finish(recordID, strings.Repeat("甲", 3*1024*1024), questionbank.Completed); err != nil {
		t.Fatal(err)
	}
	if _, err = store.SealCurrent(); err != nil {
		t.Fatal(err)
	}
	session, _ := store.Snapshot()
	if _, err = a.GetQuestionBankSessionDetail(session.ID); err == nil || !strings.Contains(err.Error(), "8 MiB") {
		t.Fatalf("超限详情未被拒绝: %v", err)
	}
}
