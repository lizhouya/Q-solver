package questionbank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Q-Solver/pkg/answersummary"
)

func TestConversationFieldsRemainOptionalForLegacyJSON(t *testing.T) {
	raw := `{"version":2,"id":"0123456789abcdef01234567","company":"旧单位","type":"综合","startedAt":"2026-10-23T00:00:00Z","state":"ended","records":[{"id":"abcdef0123456789abcdef01","createdAt":"2026-10-23T00:00:00Z","images":["assets/a.png"],"model":"m","scene":"s","answer":"a","status":"completed"}]}`
	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		t.Fatal(err)
	}
	if session.Conversation != nil || session.Records[0].KeepContext != nil || len(session.Exports) != 0 {
		t.Fatalf("旧 JSON 产生了伪会话元数据: %+v", session)
	}
}

func TestLegacyConversationRebuildUsesCompletedAnswersOnlyAndTrims(t *testing.T) {
	s := automaticBank(t)
	completedIDs := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		completedIDs = append(completedIDs, add(t, s, Completed))
	}
	add(t, s, Failed)
	add(t, s, Cancelled)
	session, _ := s.Snapshot()
	snapshot, err := s.ConversationSnapshotByID(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Quality != ConversationQualityReconstructed || len(snapshot.Turns) != MaxConversationTurns {
		t.Fatalf("旧分段重建结果不正确: %+v", snapshot)
	}
	if snapshot.Turns[0].RecordID != completedIDs[2] || snapshot.Turns[len(snapshot.Turns)-1].RecordID != completedIDs[11] {
		t.Fatalf("重建裁剪顺序不正确: %+v", snapshot.Turns)
	}
	resolved, err := s.ResolveConversation(*snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != MaxConversationTurns || len(resolved[0].Images) != 3 || resolved[0].Answer != "原答案 **A**" {
		t.Fatalf("重建内容不完整: %+v", resolved[0])
	}
}

func TestActivateHistoricalSessionPausesCurrentAndCanSwitchBack(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	first, _ := s.Snapshot()
	if err := s.End(); err != nil {
		t.Fatal(err)
	}

	add(t, s, Completed)
	second, _ := s.Snapshot()
	if first.ID == second.ID || second.State != Active {
		t.Fatalf("未创建第二个活动分段: first=%s second=%+v", first.ID, second)
	}

	result, err := s.ActivateHistoricalSession(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Previous.ID != second.ID || result.Current.ID != first.ID || result.Current.State != Active {
		t.Fatalf("切换结果不正确: %+v", result)
	}
	pausedSecond, err := s.SnapshotByID(second.ID)
	if err != nil || pausedSecond.State != Paused {
		t.Fatalf("原活动段未暂停: session=%+v err=%v", pausedSecond, err)
	}

	if _, err = s.ActivateHistoricalSession(second.ID); err != nil {
		t.Fatal(err)
	}
	pausedFirst, _ := s.SnapshotByID(first.ID)
	current, dirty := s.Snapshot()
	if dirty || pausedFirst.State != Paused || current.ID != second.ID || current.State != Active {
		t.Fatalf("切回 paused 分段失败: first=%+v current=%+v dirty=%v", pausedFirst, current, dirty)
	}
}

func TestActivateHistoricalSessionRejectsDiscardedAndRunning(t *testing.T) {
	discarded := automaticBank(t)
	add(t, discarded, Completed)
	discardedID := snapshotID(discarded)
	if err := discarded.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	if _, err := discarded.ActivateHistoricalSession(discardedID); err == nil || !strings.Contains(err.Error(), "永久只读") {
		t.Fatalf("discarded 被恢复: %v", err)
	}

	running := automaticBank(t)
	id, err := running.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil || id == "" {
		t.Fatal(err)
	}
	if _, err = running.ActivateHistoricalSession(snapshotID(running)); err == nil {
		t.Fatal("running 分段允许切换")
	}
}

func TestFinishWithConversationPersistsRecordAndSnapshotTogether(t *testing.T) {
	s := automaticBank(t)
	recordID, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	session, _ := s.Snapshot()
	snapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: session.ID, RecordID: recordID}},
		UpdatedAt: time.Now(),
	}
	if err = s.FinishWithConversation(recordID, "最终答案", Completed, snapshot); err != nil {
		t.Fatal(err)
	}
	after, dirty := s.Snapshot()
	if dirty || after.Records[0].Status != Completed || after.Records[0].Answer != "最终答案" || len(after.Records[0].AnswerItems) != 0 || after.Conversation == nil || len(after.Conversation.Turns) != 1 {
		t.Fatalf("Record 与会话快照未一起保存: %+v dirty=%v", after, dirty)
	}
	persisted, err := s.SnapshotByID(after.ID)
	if err != nil || persisted.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("磁盘快照不正确: %+v err=%v", persisted, err)
	}
}

func TestConversationSnapshotValidationRejectsSystemLikeOrDuplicateReferences(t *testing.T) {
	ref := ConversationTurnRef{SessionID: "0123456789abcdef01234567", RecordID: "abcdef0123456789abcdef01"}
	snapshot := &ConversationSnapshot{Version: 1, Quality: ConversationQualityPersisted, Turns: []ConversationTurnRef{ref, ref}}
	if err := validateConversationSnapshot(snapshot); err == nil {
		t.Fatal("重复引用被接受")
	}
	snapshot = &ConversationSnapshot{Version: 1, Quality: "unknown"}
	if err := validateConversationSnapshot(snapshot); err == nil {
		t.Fatal("未知质量标记被接受")
	}
}

func TestBeginWithContextPersistsBoundary(t *testing.T) {
	s := automaticBank(t)
	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
	if err != nil {
		t.Fatal(err)
	}
	session, _ := s.Snapshot()
	record := findRecord(session, recordID)
	if record == nil || record.KeepContext == nil || *record.KeepContext {
		t.Fatalf("KeepContext=false 边界未持久化: %+v", record)
	}
}

func TestAppendExportSnapshotNormalizesLegacyAndKeepsVersions(t *testing.T) {
	legacyAt := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	newAt := legacyAt.Add(time.Hour)
	session := &Session{
		ExportedAt:      &legacyAt,
		ExportDirectory: `C:\旧导出`,
	}
	appendExportSnapshot(session, newAt, `C:\新导出`)
	if len(session.Exports) != 2 {
		t.Fatalf("导出历史数量错误: %+v", session.Exports)
	}
	if session.Exports[0].Directory != `C:\旧导出` || session.Exports[1].Directory != `C:\新导出` {
		t.Fatalf("导出历史顺序错误: %+v", session.Exports)
	}
	appendExportSnapshot(session, newAt, `C:\新导出`)
	if len(session.Exports) != 2 {
		t.Fatalf("重复导出历史未去重: %+v", session.Exports)
	}
}

func activationFailureBank(t *testing.T) (*Store, *Session, *Session) {
	t.Helper()
	s := automaticBank(t)
	add(t, s, Completed)
	target, _ := s.Snapshot()
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	add(t, s, Completed)
	source, _ := s.Snapshot()
	return s, source, target
}

func diskSession(t *testing.T, s *Store, id string) *Session {
	t.Helper()
	base, ok := s.findSessionBase(id)
	if !ok {
		t.Fatalf("磁盘分段不存在: %s", id)
	}
	session, err := s.readSessionFromBase(base, id)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestActivateHistoricalSessionSourcePauseFailureKeepsState(t *testing.T) {
	s, source, target := activationFailureBank(t)
	s.saveSessionHook = func(session *Session, _ string) error {
		if session.ID == source.ID && session.State == Paused {
			return errors.New("synthetic source pause failure")
		}
		return nil
	}

	_, err := s.ActivateHistoricalSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "暂停当前分段失败") {
		t.Fatalf("源暂停失败未正确返回: %v", err)
	}
	current, dirty := s.Snapshot()
	if dirty || current.ID != source.ID || current.State != Active {
		t.Fatalf("源暂停失败后内存状态变化: current=%+v dirty=%v", current, dirty)
	}
	if got := diskSession(t, s, source.ID); got.State != Active {
		t.Fatalf("源暂停失败后磁盘状态变化: %+v", got)
	}
	if got := diskSession(t, s, target.ID); got.State != Ended {
		t.Fatalf("源暂停失败后目标状态变化: %+v", got)
	}
	if s.writeBlocked != nil {
		t.Fatalf("首次写失败不应永久阻断 Store: %v", s.writeBlocked)
	}
}

func TestActivateHistoricalSessionTargetFailureRollsBackSource(t *testing.T) {
	s, source, target := activationFailureBank(t)
	s.saveSessionHook = func(session *Session, _ string) error {
		if session.ID == target.ID && session.State == Active {
			return errors.New("synthetic target activation failure")
		}
		return nil
	}

	_, err := s.ActivateHistoricalSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "激活目标分段失败") {
		t.Fatalf("目标激活失败未正确返回: %v", err)
	}
	current, dirty := s.Snapshot()
	if dirty || current.ID != source.ID || current.State != Active {
		t.Fatalf("补偿成功后内存未恢复: current=%+v dirty=%v", current, dirty)
	}
	if got := diskSession(t, s, source.ID); got.State != Active {
		t.Fatalf("补偿成功后源磁盘状态错误: %+v", got)
	}
	if got := diskSession(t, s, target.ID); got.State != Ended {
		t.Fatalf("目标失败后目标磁盘状态错误: %+v", got)
	}
	if s.writeBlocked != nil {
		t.Fatalf("补偿成功不应阻断后续写入: %v", s.writeBlocked)
	}
}

func TestActivateHistoricalSessionRollbackFailureBlocksAllWrites(t *testing.T) {
	s, source, target := activationFailureBank(t)
	s.saveSessionHook = func(session *Session, _ string) error {
		if session.ID == target.ID && session.State == Active {
			return errors.New("synthetic target activation failure")
		}
		if session.ID == source.ID && session.State == Active {
			return errors.New("synthetic source rollback failure")
		}
		return nil
	}

	_, err := s.ActivateHistoricalSession(target.ID)
	if err == nil || !strings.Contains(err.Error(), "恢复原分段也失败") {
		t.Fatalf("双重写失败未完整报告: %v", err)
	}
	current, dirty := s.Snapshot()
	if dirty || current.ID != source.ID || current.State != Paused {
		t.Fatalf("补偿失败后内存未对齐磁盘保守状态: current=%+v dirty=%v", current, dirty)
	}
	if got := diskSession(t, s, source.ID); got.State != Paused {
		t.Fatalf("补偿失败后源磁盘状态错误: %+v", got)
	}
	if got := diskSession(t, s, target.ID); got.State != Ended {
		t.Fatalf("补偿失败后目标磁盘状态错误: %+v", got)
	}
	if s.writeBlocked == nil {
		t.Fatal("补偿失败后 Store 未进入写阻断")
	}

	operations := map[string]func() error{
		"set-default-company": func() error { return s.SetDefaultCompany("新单位") },
		"start":               func() error { return s.Start("新单位", "综合") },
		"begin": func() error {
			_, opErr := s.Begin([]string{shot(t, false)}, "model", "scene")
			return opErr
		},
		"append":  func() error { return s.Append(source.Records[0].ID, "late") },
		"finish":  func() error { return s.Finish(source.Records[0].ID, "late", Completed) },
		"seal":    func() error { _, opErr := s.SealCurrent(); return opErr },
		"discard": func() error { return s.DiscardCurrent() },
		"resume":  func() error { return s.Resume() },
		"flush":   func() error { return s.Flush() },
		"open":    func() error { return s.Open(source.ID) },
		"metadata": func() error {
			return s.UpdateSessionMetadata(target.ID, "名称", "单位", "出题方", "题型", true)
		},
		"organize": func() error {
			return s.OrganizeSession(context.Background(), target.ID, nil, "model", nil)
		},
		"export": func() error {
			_, opErr := s.ExportSession(target.ID, ExportOptions{Name: "blocked"})
			return opErr
		},
		"activate": func() error { _, opErr := s.ActivateHistoricalSession(target.ID); return opErr },
		"finish-with-conversation": func() error {
			return s.FinishWithConversation(source.Records[0].ID, "late", Completed, &ConversationSnapshot{
				Version: 1, Quality: ConversationQualityPersisted,
			})
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			opErr := operation()
			if opErr == nil || !strings.Contains(opErr.Error(), "题库写入已阻止") {
				t.Fatalf("%s 未被统一写阻断: %v", name, opErr)
			}
		})
	}

	if _, listErr := s.List(); listErr != nil {
		t.Fatalf("写阻断后 List 不应受影响: %v", listErr)
	}
	if _, imageErr := s.ReadRecordImage(source.ID, source.Records[0].ID, 0); imageErr != nil {
		t.Fatalf("写阻断后截图读取不应受影响: %v", imageErr)
	}
}

func TestActivateHistoricalSessionReactivatesCurrentPausedSession(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	current, _ := s.Snapshot()

	reopened, err := NewStore(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Open(current.ID); err != nil {
		t.Fatal(err)
	}
	paused, dirty := reopened.Snapshot()
	if dirty || paused.State != Paused {
		t.Fatalf("重开活动段后未进入 paused: %+v dirty=%v", paused, dirty)
	}

	result, err := reopened.ActivateHistoricalSession(current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Previous.ID != current.ID || result.Previous.State != Paused || result.Current.ID != current.ID || result.Current.State != Active {
		t.Fatalf("same-current paused→active 结果错误: %+v", result)
	}
	persisted := diskSession(t, reopened, current.ID)
	if persisted.State != Active {
		t.Fatalf("same-current 激活未持久化: %+v", persisted)
	}
}

func TestActivateHistoricalExportedSessionPreservesExportHistory(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	targetID, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	exported, err := s.ExportSession(targetID, ExportOptions{Name: "历史导出", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.SnapshotByID(targetID)
	if err != nil {
		t.Fatal(err)
	}
	if before.State != Exported || len(before.Exports) != 1 {
		t.Fatalf("测试前置导出状态错误: %+v", before)
	}

	add(t, s, Completed)
	if _, err = s.ActivateHistoricalSession(targetID); err != nil {
		t.Fatal(err)
	}
	after, err := s.SnapshotByID(targetID)
	if err != nil {
		t.Fatal(err)
	}
	if after.State != Active || after.ExportDirectory != exported.Directory || after.ExportedAt == nil || len(after.Exports) != 1 || after.Exports[0].Directory != exported.Directory {
		t.Fatalf("激活 exported 分段破坏了导出元数据: before=%+v after=%+v", before, after)
	}
}

func TestFinishWithConversationRetryCommitsPendingCompletionAtomically(t *testing.T) {
	s := automaticBank(t)
	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", true)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	snapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: current.ID, RecordID: recordID}},
		UpdatedAt: time.Now(),
	}
	failed := false
	s.saveSessionHook = func(session *Session, _ string) error {
		record := findRecord(session, recordID)
		if !failed && record != nil && record.Status == Completed {
			failed = true
			return errors.New("synthetic completion failure")
		}
		return nil
	}

	if err = s.FinishWithConversation(recordID, "最终答案", Completed, snapshot); err == nil {
		t.Fatal("完成态写入失败未返回错误")
	}
	beforeRetry, dirty := s.Snapshot()
	if !dirty || beforeRetry.Records[0].Status != Running || beforeRetry.Conversation != nil {
		t.Fatalf("失败完成态不应伪装为已提交: current=%+v dirty=%v", beforeRetry, dirty)
	}
	if s.pendingCompletion == nil {
		t.Fatal("失败完成态未保留为可重试副本")
	}
	if _, beginErr := s.Begin([]string{shot(t, false)}, "model", "scene"); beginErr == nil || !strings.Contains(beginErr.Error(), "完成结果等待重试保存") {
		t.Fatalf("待重试完成态期间仍允许新写入: %v", beginErr)
	}

	s.saveSessionHook = nil
	if err = s.Flush(); err != nil {
		t.Fatalf("重试完成态保存失败: %v", err)
	}
	afterRetry, dirty := s.Snapshot()
	if dirty || s.pendingCompletion != nil || afterRetry.Records[0].Status != Completed || afterRetry.Records[0].Answer != "最终答案" || afterRetry.Conversation == nil {
		t.Fatalf("重试未原子提交完成态: current=%+v dirty=%v pending=%+v", afterRetry, dirty, s.pendingCompletion)
	}
	persisted := diskSession(t, s, current.ID)
	if persisted.Records[0].Status != Completed || persisted.Records[0].Answer != "最终答案" || persisted.Conversation == nil || persisted.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("磁盘完成态不完整: %+v", persisted)
	}
}

func TestContinueExportedSessionAppendsAndCreatesNewExportVersion(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	targetID, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.ExportSession(targetID, ExportOptions{Name: "版本题库", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	originalRaw, err := os.ReadFile(first.RawMarkdown)
	if err != nil {
		t.Fatal(err)
	}

	add(t, s, Completed)
	if _, err = s.ActivateHistoricalSession(targetID); err != nil {
		t.Fatal(err)
	}
	newRecord := add(t, s, Completed)
	active, _ := s.Snapshot()
	if active.ID != targetID || findRecord(active, newRecord) == nil || len(active.Records) != 2 {
		t.Fatalf("新题未追加到续接的 exported 分段: %+v", active)
	}
	if _, err = s.SealCurrent(); err != nil {
		t.Fatal(err)
	}
	second, err := s.ExportSession(targetID, ExportOptions{Name: "版本题库", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if second.Directory == first.Directory || filepath.Base(second.Directory) != filepath.Base(first.Directory)+"_02" {
		t.Fatalf("再次导出未创建新版本目录: first=%s second=%s", first.Directory, second.Directory)
	}
	unchangedRaw, err := os.ReadFile(first.RawMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchangedRaw) != string(originalRaw) {
		t.Fatal("再次导出修改了旧导出目录内容")
	}
	persisted, err := s.SnapshotByID(targetID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.State != Exported || len(persisted.Exports) != 2 || persisted.Exports[0].Directory != first.Directory || persisted.Exports[1].Directory != second.Directory {
		t.Fatalf("再次导出历史不完整: %+v", persisted.Exports)
	}
}

func readyAnswerItem(label, value string) answersummary.Item {
	return answersummary.Item{
		Label:  label,
		Kind:   answersummary.KindOther,
		Value:  value,
		Status: answersummary.StatusReady,
	}
}

func finishWithAnswerItems(t *testing.T, s *Store, keepContext bool, answer string, items []answersummary.Item) string {
	t.Helper()
	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", keepContext)
	if err != nil {
		t.Fatal(err)
	}
	session, _ := s.Snapshot()
	snapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: session.ID, RecordID: recordID}},
		UpdatedAt: time.Now(),
	}
	if err = s.FinishWithConversationResult(recordID, answer, items, Completed, snapshot); err != nil {
		t.Fatal(err)
	}
	return recordID
}

func TestFinishWithConversationResultSavesAnswerItemsAndSnapshotOnce(t *testing.T) {
	s := automaticBank(t)
	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.Snapshot()
	items := []answersummary.Item{
		readyAnswerItem("12", "B"),
		readyAnswerItem("13", "AC"),
	}
	snapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: before.ID, RecordID: recordID}},
		UpdatedAt: time.Now(),
	}

	saveCalls := 0
	candidateWasComplete := false
	currentStayedRunning := false
	s.saveSessionHook = func(candidate *Session, _ string) error {
		saveCalls++
		record := findRecord(candidate, recordID)
		candidateWasComplete = record != nil && record.Answer == "完整回答" && record.Status == Completed &&
			len(record.AnswerItems) == 2 && record.AnswerItems[0] == items[0] &&
			candidate.Conversation != nil && len(candidate.Conversation.Turns) == 1
		current := findRecord(s.current, recordID)
		currentStayedRunning = current != nil && current.Status == Running && current.Answer == "" &&
			len(current.AnswerItems) == 0 && s.current.Conversation == nil
		return nil
	}

	if err = s.FinishWithConversationResult(recordID, "完整回答", items, Completed, snapshot); err != nil {
		t.Fatal(err)
	}
	if saveCalls != 1 || !candidateWasComplete || !currentStayedRunning {
		t.Fatalf("完成结果未通过单一完整候选保存: calls=%d candidate=%v current=%v", saveCalls, candidateWasComplete, currentStayedRunning)
	}

	items[0].Label = "caller-mutated"
	snapshot.Turns[0].RecordID = "caller-mutated"
	current, dirty := s.Snapshot()
	record := findRecord(current, recordID)
	if dirty || current.Version != currentSessionVersion || record == nil || record.Answer != "完整回答" ||
		record.Status != Completed || len(record.AnswerItems) != 2 || record.AnswerItems[0].Label != "12" ||
		current.Conversation == nil || current.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("内存完成结果不完整或受调用方修改影响: current=%+v record=%+v dirty=%v", current, record, dirty)
	}
	persisted := diskSession(t, s, current.ID)
	persistedRecord := findRecord(persisted, recordID)
	if persisted.Version != 2 || persistedRecord == nil || persistedRecord.Answer != "完整回答" ||
		len(persistedRecord.AnswerItems) != 2 || persistedRecord.AnswerItems[1].Label != "13" ||
		persisted.Conversation == nil || persisted.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("磁盘完成结果不完整: session=%+v record=%+v", persisted, persistedRecord)
	}
}

func TestFinishWithConversationResultRetryPreservesAllFieldsAtomically(t *testing.T) {
	s := automaticBank(t)
	recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", true)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	items := []answersummary.Item{readyAnswerItem("8", "SECRET-VALUE")}
	snapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: current.ID, RecordID: recordID}},
		UpdatedAt: time.Now(),
	}
	failed := false
	s.saveSessionHook = func(candidate *Session, _ string) error {
		record := findRecord(candidate, recordID)
		if !failed && record != nil && record.Status == Completed {
			failed = true
			return errors.New("synthetic result failure")
		}
		return nil
	}

	if err = s.FinishWithConversationResult(recordID, "完整回答", items, Completed, snapshot); err == nil {
		t.Fatal("完成结果写入失败未返回错误")
	}
	beforeRetry, dirty := s.Snapshot()
	beforeRecord := findRecord(beforeRetry, recordID)
	if !dirty || beforeRecord == nil || beforeRecord.Status != Running || beforeRecord.Answer != "" ||
		len(beforeRecord.AnswerItems) != 0 || beforeRetry.Conversation != nil {
		t.Fatalf("失败后 current 出现半完成状态: current=%+v record=%+v dirty=%v", beforeRetry, beforeRecord, dirty)
	}
	pendingRecord := findRecord(s.pendingCompletion, recordID)
	if pendingRecord == nil || pendingRecord.Answer != "完整回答" || pendingRecord.Status != Completed ||
		len(pendingRecord.AnswerItems) != 1 || pendingRecord.AnswerItems[0] != items[0] ||
		s.pendingCompletion.Conversation == nil || s.pendingCompletion.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("pending completion 未保留完整结果: pending=%+v record=%+v", s.pendingCompletion, pendingRecord)
	}

	items[0].Label = "caller-mutated"
	snapshot.Turns[0].RecordID = "caller-mutated"
	s.saveSessionHook = nil
	if err = s.Flush(); err != nil {
		t.Fatalf("重试保存失败: %v", err)
	}
	afterRetry, dirty := s.Snapshot()
	afterRecord := findRecord(afterRetry, recordID)
	if dirty || s.pendingCompletion != nil || afterRecord == nil || afterRecord.Status != Completed ||
		afterRecord.Answer != "完整回答" || len(afterRecord.AnswerItems) != 1 ||
		afterRetry.Conversation == nil || afterRetry.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("重试后完成结果不完整: current=%+v record=%+v dirty=%v pending=%+v", afterRetry, afterRecord, dirty, s.pendingCompletion)
	}
	persisted := diskSession(t, s, current.ID)
	persistedRecord := findRecord(persisted, recordID)
	if persistedRecord == nil || persistedRecord.Answer != "完整回答" || len(persistedRecord.AnswerItems) != 1 ||
		persisted.Conversation == nil || persisted.Conversation.Turns[0].RecordID != recordID {
		t.Fatalf("重试后磁盘结果不完整: session=%+v record=%+v", persisted, persistedRecord)
	}
}

func TestCurrentNumberingContextUsesValidatedItemsThenLegacyWithoutAnswers(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, true, `## 最终答案汇总
| 题号 | 旧答案 |
|---|---|
| legacy-1 | LEGACY-SECRET |`, []answersummary.Item{readyAnswerItem("1", "STRUCTURED-SECRET")})

	secondID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	secondSnapshot := &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityPersisted,
		Turns:     []ConversationTurnRef{{SessionID: current.ID, RecordID: secondID}},
		UpdatedAt: time.Now(),
	}
	if err = s.FinishWithConversation(secondID, "## 最终答案汇总\n2：LEGACY-FALLBACK-SECRET", Completed, secondSnapshot); err != nil {
		t.Fatal(err)
	}
	findRecord(s.current, secondID).AnswerItems = []answersummary.Item{{Label: "invalid-item-label", Kind: "invalid", Value: "INVALID-SECRET", Status: answersummary.StatusReady}}
	finishWithAnswerItems(t, s, true, "malformed label", []answersummary.Item{readyAnswerItem("答案：LABEL-SECRET", "value")})

	failedID, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Append(failedID, "## 最终答案汇总\n999：FAILED-SECRET"); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(failedID, "ignored", Failed); err != nil {
		t.Fatal(err)
	}

	complexID := finishWithAnswerItems(t, s, false, "完整回答", []answersummary.Item{readyAnswerItem("第二章-3", "COMPLEX-SECRET")})
	complexRecord := findRecord(s.current, complexID)
	if complexRecord == nil || complexRecord.KeepContext == nil || *complexRecord.KeepContext {
		t.Fatalf("KeepContext=false 测试前置错误: %+v", complexRecord)
	}
	runningID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := s.CurrentNumberingContext(runningID)
	want := []string{"1", "2", "第二章-3"}
	if fmt.Sprint(ctx.PreviousLabels) != fmt.Sprint(want) || ctx.SuggestedNextLabel != "" {
		t.Fatalf("编号上下文错误: got=%+v want labels=%v", ctx, want)
	}
	joined := strings.Join(ctx.PreviousLabels, "\n") + "\n" + ctx.SuggestedNextLabel
	for _, forbidden := range []string{"STRUCTURED-SECRET", "LEGACY-SECRET", "LEGACY-FALLBACK-SECRET", "INVALID-SECRET", "LABEL-SECRET", "FAILED-SECRET", "999", "invalid-item-label", "答案"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("编号上下文泄漏答案或无效记录 %q: %+v", forbidden, ctx)
		}
	}
}

func TestCurrentNumberingContextOnlyUsesRecordsEarlierThanExcluded(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, true, "first", []answersummary.Item{readyAnswerItem("1", "A")})
	excludedID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", true)
	if err != nil {
		t.Fatal(err)
	}
	finishWithAnswerItems(t, s, true, "later", []answersummary.Item{readyAnswerItem("99", "B")})

	ctx := s.CurrentNumberingContext(excludedID)
	if fmt.Sprint(ctx.PreviousLabels) != "[1]" || ctx.SuggestedNextLabel != "2" {
		t.Fatalf("排除记录之后的 completed 记录进入上下文: %+v", ctx)
	}
}

func TestCurrentNumberingContextKeepsRecent64UniqueAndSafeSuggestion(t *testing.T) {
	s := automaticBank(t)
	first := make([]answersummary.Item, 0, 50)
	for i := 1; i <= 50; i++ {
		first = append(first, readyAnswerItem(fmt.Sprint(i), fmt.Sprintf("value-%d", i)))
	}
	finishWithAnswerItems(t, s, true, "first batch", first)
	second := make([]answersummary.Item, 0, 26)
	for i := 45; i <= 70; i++ {
		second = append(second, readyAnswerItem(fmt.Sprint(i), fmt.Sprintf("other-%d", i)))
	}
	finishWithAnswerItems(t, s, true, "second batch", second)
	runningID, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}

	ctx := s.CurrentNumberingContext(runningID)
	if len(ctx.PreviousLabels) != 64 || ctx.PreviousLabels[0] != "7" || ctx.PreviousLabels[63] != "70" || ctx.SuggestedNextLabel != "71" {
		t.Fatalf("最近 64 个去重标签或数字建议错误: %+v", ctx)
	}

	overflow := automaticBank(t)
	finishWithAnswerItems(t, overflow, true, "overflow", []answersummary.Item{readyAnswerItem("18446744073709551615", "value")})
	overflowRunning, err := overflow.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if got := overflow.CurrentNumberingContext(overflowRunning); got.SuggestedNextLabel != "" {
		t.Fatalf("溢出数字不应猜测下一编号: %+v", got)
	}
}

func TestCurrentNumberingContextNewSessionDoesNotInheritLabels(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, true, "old", []answersummary.Item{readyAnswerItem("7", "value")})
	oldSession, _ := s.Snapshot()
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	newRecordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
	if err != nil {
		t.Fatal(err)
	}
	newSession, _ := s.Snapshot()
	if newSession.ID == oldSession.ID {
		t.Fatal("未创建新 Session")
	}
	if got := s.CurrentNumberingContext(newRecordID); len(got.PreviousLabels) != 0 || got.SuggestedNextLabel != "" {
		t.Fatalf("新 Session 继承了旧编号: %+v", got)
	}
}

func TestCurrentNumberingContextReadsLegacyV1V2WithoutRewriting(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			s := automaticBank(t)
			recordID, err := s.BeginWithContext([]string{shot(t, false)}, "model", "scene", false)
			if err != nil {
				t.Fatal(err)
			}
			session, _ := s.Snapshot()
			snapshot := &ConversationSnapshot{
				Version:   1,
				Quality:   ConversationQualityPersisted,
				Turns:     []ConversationTurnRef{{SessionID: session.ID, RecordID: recordID}},
				UpdatedAt: time.Now(),
			}
			if err = s.FinishWithConversation(recordID, "## 最终答案汇总\n12：LEGACY-ANSWER", Completed, snapshot); err != nil {
				t.Fatal(err)
			}
			if err = s.End(); err != nil {
				t.Fatal(err)
			}
			manifest := filepath.Join(s.sessionDir(session.ID), "session.json")
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = json.Unmarshal(raw, &document); err != nil {
				t.Fatal(err)
			}
			document["version"] = version
			records := document["records"].([]any)
			delete(records[0].(map[string]any), "answerItems")
			legacyRaw, err := json.MarshalIndent(document, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			legacyRaw = append(legacyRaw, '\n')
			if err = os.WriteFile(manifest, legacyRaw, 0600); err != nil {
				t.Fatal(err)
			}

			reopened, err := NewStore(s.Root())
			if err != nil {
				t.Fatal(err)
			}
			if err = reopened.Open(session.ID); err != nil {
				t.Fatal(err)
			}
			ctx := reopened.CurrentNumberingContext("")
			if fmt.Sprint(ctx.PreviousLabels) != "[12]" || ctx.SuggestedNextLabel != "13" {
				t.Fatalf("旧 v%d JSON 编号恢复错误: %+v", version, ctx)
			}
			after, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, legacyRaw) {
				t.Fatalf("只读打开/编号查询改写了旧 v%d JSON", version)
			}
		})
	}
}

func TestCurrentNumberingContextSurvivesActiveReopen(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, false, "active", []answersummary.Item{readyAnswerItem("15", "value")})
	original, _ := s.Snapshot()

	reopened, err := NewStore(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Open(original.ID); err != nil {
		t.Fatal(err)
	}
	paused, dirty := reopened.Snapshot()
	if dirty || paused.State != Paused {
		t.Fatalf("活动 Session reopen 后状态错误: session=%+v dirty=%v", paused, dirty)
	}
	manifest := filepath.Join(reopened.sessionDir(original.ID), "session.json")
	before, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	ctx := reopened.CurrentNumberingContext("")
	if fmt.Sprint(ctx.PreviousLabels) != "[15]" || ctx.SuggestedNextLabel != "16" {
		t.Fatalf("reopen 后编号丢失: %+v", ctx)
	}
	after, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("编号只读查询改写了 reopen 后的 Session")
	}
}

func TestCurrentNumberingContextSurvivesEndedContinuation(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, false, "ended", []answersummary.Item{readyAnswerItem("30", "value")})
	target, _ := s.Snapshot()
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	finishWithAnswerItems(t, s, true, "other session", []answersummary.Item{readyAnswerItem("100", "other")})
	if _, err := s.ActivateHistoricalSession(target.ID); err != nil {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	if current.ID != target.ID || current.State != Active {
		t.Fatalf("ended Session 未真正续接: %+v", current)
	}
	ctx := s.CurrentNumberingContext("")
	if fmt.Sprint(ctx.PreviousLabels) != "[30]" || ctx.SuggestedNextLabel != "31" {
		t.Fatalf("ended 续接丢失编号: %+v", ctx)
	}
}

func TestCurrentNumberingContextSurvivesExportedContinuation(t *testing.T) {
	s := automaticBank(t)
	finishWithAnswerItems(t, s, false, "exported", []answersummary.Item{readyAnswerItem("20", "value")})
	targetID, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ExportSession(targetID, ExportOptions{Name: "编号续接", RawOnly: true}); err != nil {
		t.Fatal(err)
	}
	finishWithAnswerItems(t, s, true, "other session", []answersummary.Item{readyAnswerItem("100", "other")})
	if _, err = s.ActivateHistoricalSession(targetID); err != nil {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	if current.ID != targetID || current.State != Active {
		t.Fatalf("exported Session 未真正续接: %+v", current)
	}
	ctx := s.CurrentNumberingContext("")
	if fmt.Sprint(ctx.PreviousLabels) != "[20]" || ctx.SuggestedNextLabel != "21" {
		t.Fatalf("exported 续接丢失编号: %+v", ctx)
	}
}

func TestClearCurrentConversationDoesNotReconstructLegacyTurns(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	before, _ := s.Snapshot()
	if err := s.ClearCurrentConversation(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.ConversationSnapshotByID(before.ID)
	if err != nil || snapshot == nil || len(snapshot.Turns) != 0 {
		t.Fatalf("cleared history rebuilt: %+v %v", snapshot, err)
	}
	reopened, err := NewStore(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = reopened.ConversationSnapshotByID(before.ID)
	if err != nil || len(snapshot.Turns) != 0 {
		t.Fatal("empty snapshot not persisted")
	}
	after, _ := s.Snapshot()
	if len(after.Records) != len(before.Records) || after.Records[0].Answer != before.Records[0].Answer {
		t.Fatal("archive altered")
	}
}

func TestClearCurrentConversationFailureRetriesEmptySnapshot(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	before, _ := s.Snapshot()
	s.saveSessionHook = func(*Session, string) error { return errors.New("synthetic failure") }
	if err := s.ClearCurrentConversation(); err == nil {
		t.Fatal("expected failure")
	}
	current, dirty := s.Snapshot()
	if !dirty || current.Conversation != nil || len(current.Records) != len(before.Records) {
		t.Fatal("failed clear mutated current")
	}
	s.saveSessionHook = nil
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.ConversationSnapshotByID(before.ID)
	if err != nil || len(snapshot.Turns) != 0 {
		t.Fatal("retry restored old history")
	}
}
