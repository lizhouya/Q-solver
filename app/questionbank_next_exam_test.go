package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/solution"
)

func nextExamApp(t *testing.T) (*App, *questionbank.Store, string, string) {
	t.Helper()
	a, store, image := archiveApp(t)
	a.solver = solution.NewSolver(nil)
	recordID, err := store.BeginWithContext([]string{image}, "model", "scene", true)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Finish(recordID, "当前套答案", questionbank.Completed); err != nil {
		t.Fatal(err)
	}
	current, _ := store.Snapshot()
	conversation := solution.ConversationState{Turns: []solution.ConversationTurn{{
		User:      llm.NewMultiPartMessage(llm.RoleUser, []llm.ContentPart{llm.ImagePart(image)}),
		Assistant: llm.NewAssistantMessage("当前套答案"),
	}}}
	if err = a.solver.ReplaceConversation(conversation); err != nil {
		t.Fatal(err)
	}
	a.conversationRefs = []questionbank.ConversationTurnRef{{SessionID: current.ID, RecordID: recordID}}
	a.conversationEpoch = 7
	a.screenshotBuffer = []string{"pending-1", "pending-2"}
	a.questionBank.message = "原状态消息"
	return a, store, image, recordID
}

func assertNextExamRuntimeUnchanged(t *testing.T, a *App, store *questionbank.Store, sessionID, recordID, sessionState string) {
	t.Helper()
	current, dirty := store.Snapshot()
	if dirty || current == nil || current.ID != sessionID || current.State != sessionState {
		t.Fatalf("拒绝后 Store 变化: current=%+v dirty=%v", current, dirty)
	}
	if a.conversationEpoch != 7 {
		t.Fatalf("拒绝后 conversationEpoch 变化: %d", a.conversationEpoch)
	}
	if len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != recordID {
		t.Fatalf("拒绝后 conversationRefs 变化: %+v", a.conversationRefs)
	}
	if len(a.screenshotBuffer) != 2 || a.screenshotBuffer[0] != "pending-1" || a.screenshotBuffer[1] != "pending-2" {
		t.Fatalf("拒绝后截图缓冲变化: %+v", a.screenshotBuffer)
	}
	conversation := a.solver.SnapshotConversation()
	if len(conversation.Turns) != 1 || conversation.Turns[0].Assistant.Content != "当前套答案" {
		t.Fatalf("拒绝后 Solver 历史变化: %+v", conversation)
	}
	if a.questionBank != nil && a.questionBank.message != "原状态消息" {
		t.Fatalf("拒绝后题库消息变化: %q", a.questionBank.message)
	}
}

func TestStartNextExamSealsThenClearsRuntimeState(t *testing.T) {
	a, store, image, _ := nextExamApp(t)
	bridge, err := mobile.NewBridge(41)
	if err != nil {
		t.Fatal(err)
	}
	bridge.Publish("solution-stream-start")
	bridge.Publish("solution-stream-chunk", "手机端既有答案")
	bridge.Publish("solution", "手机端既有答案")
	a.mobileBridge = bridge
	a.mobileRuntimeState = config.MobileRuntimeState{StableLink: true, Token: "synthetic-token", ClearSeq: 41, Generation: 3}
	a.mobileStateReady = true
	a.mobileStatus = mobileStatusReady
	mobileSnapshotBefore := bridge.GetState()
	mobileRuntimeBefore := a.mobileRuntimeState
	mobileEpochBefore := a.mobileDisplayEpoch()
	before, _ := store.Snapshot()

	result, err := a.StartNextExam(true)
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != before.ID || result.PreviousState != questionbank.Active || result.SessionState != questionbank.Ended || !result.Sealed {
		t.Fatalf("题库结果错误: %+v", result)
	}
	if result.ConversationEpoch != 8 || result.ClearedScreenshotCount != 2 {
		t.Fatalf("运行时清理结果错误: %+v", result)
	}
	after, dirty := store.Snapshot()
	if dirty || after.ID != before.ID || after.State != questionbank.Ended || after.EndedAt == nil {
		t.Fatalf("当前分段未封存: session=%+v dirty=%v", after, dirty)
	}
	if len(a.solver.SnapshotConversation().Turns) != 0 || len(a.conversationRefs) != 0 || len(a.screenshotBuffer) != 0 || a.conversationEpoch != 8 {
		t.Fatalf("成功后运行时状态未清空: refs=%+v screenshots=%+v epoch=%d conversation=%+v", a.conversationRefs, a.screenshotBuffer, a.conversationEpoch, a.solver.SnapshotConversation())
	}
	if a.mobileDisplayEpoch() != mobileEpochBefore || a.mobileRuntimeState != mobileRuntimeBefore || !reflect.DeepEqual(bridge.GetState(), mobileSnapshotBefore) || !a.mobileStateReady || a.mobileStatus != mobileStatusReady {
		t.Fatalf("开始下一套不应修改 Mobile: epoch=%d runtime=%+v before=%+v after=%+v", a.mobileDisplayEpoch(), a.mobileRuntimeState, mobileSnapshotBefore, bridge.GetState())
	}

	nextRecordID, err := store.BeginWithContext([]string{image}, "next-model", "next-scene", true)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := store.Snapshot()
	if nextRecordID == "" || next.ID == before.ID || next.State != questionbank.Active || len(next.Records) != 1 {
		t.Fatalf("下一次 BeginWithContext 未懒创建新分段: before=%s next=%+v", before.ID, next)
	}
}

func TestStartNextExamRequiresConfirmationWithoutSideEffects(t *testing.T) {
	a, store, _, recordID := nextExamApp(t)
	before, _ := store.Snapshot()

	result, err := a.StartNextExam(false)
	if err == nil || !strings.Contains(err.Error(), "确认") {
		t.Fatalf("未确认时未拒绝: result=%+v err=%v", result, err)
	}
	assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
}

func TestStartNextExamRejectsBusyAndUnavailableStatesWithoutSideEffects(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*App)
		want    string
		cleanup func(*App)
	}{
		{
			name: "in-flight task",
			prepare: func(a *App) {
				_, _ = a.taskManager.StartTask("solve")
			},
			want: "完全结束",
			cleanup: func(a *App) {
				a.taskManager.CompleteTask(a.taskManager.GetCurrentTaskID())
			},
		},
		{
			name: "cancelled but incomplete task",
			prepare: func(a *App) {
				_, _ = a.taskManager.StartTask("solve")
				if !a.taskManager.CancelCurrentTask() {
					panic("测试前置取消任务失败")
				}
			},
			want: "完全结束",
			cleanup: func(a *App) {
				a.taskManager.CompleteTask(a.taskManager.GetCurrentTaskID())
			},
		},
		{name: "capture", prepare: func(a *App) { a.questionBank.captures = 1 }, want: "题目"},
		{name: "organizing", prepare: func(a *App) {
			a.questionBank.organizing = true
			a.questionBank.operation = "organizing"
		}, want: "整理或导出"},
		{name: "exporting", prepare: func(a *App) {
			a.questionBank.operation = "exporting"
		}, want: "整理或导出"},
		{name: "closing", prepare: func(a *App) { a.questionBank.closing = true }, want: "应用正在关闭"},
		{name: "solver unavailable", prepare: func(a *App) { a.solver = nil }, want: "尚未就绪"},
		{name: "task coordinator unavailable", prepare: func(a *App) { a.taskManager = nil }, want: "尚未就绪"},
		{name: "question bank unavailable", prepare: func(a *App) { a.questionBank = nil }, want: "题库服务不可用"},
		{name: "store unavailable", prepare: func(a *App) { a.questionBank.store = nil }, want: "选择题库目录"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, store, _, recordID := nextExamApp(t)
			before, _ := store.Snapshot()
			originalSolver := a.solver
			originalTaskManager := a.taskManager
			originalQuestionBank := a.questionBank
			originalStore := a.questionBank.store
			tt.prepare(a)

			result, err := a.StartNextExam(true)
			if tt.cleanup != nil {
				tt.cleanup(a)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("未按预期拒绝: result=%+v err=%v", result, err)
			}

			if a.solver == nil {
				a.solver = originalSolver
			}
			if a.taskManager == nil {
				a.taskManager = originalTaskManager
			}
			if a.questionBank == nil {
				a.questionBank = originalQuestionBank
			}
			if a.questionBank.store == nil {
				a.questionBank.store = originalStore
			}
			assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
		})
	}
}

func TestStartNextExamSealsPausedSession(t *testing.T) {
	a, store, _, _ := nextExamApp(t)
	current, _ := store.Snapshot()
	reopened, err := questionbank.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Open(current.ID); err != nil {
		t.Fatal(err)
	}
	paused, dirty := reopened.Snapshot()
	if dirty || paused.State != questionbank.Paused {
		t.Fatalf("测试前置未得到 paused 分段: %+v dirty=%v", paused, dirty)
	}
	a.questionBank.store = reopened

	result, err := a.StartNextExam(true)
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != current.ID || result.PreviousState != questionbank.Paused || result.SessionState != questionbank.Ended || !result.Sealed {
		t.Fatalf("paused App 封存结果错误: %+v", result)
	}
	sealed, dirty := reopened.Snapshot()
	if dirty || sealed.State != questionbank.Ended || sealed.EndedAt == nil {
		t.Fatalf("paused App 封存未落盘: %+v dirty=%v", sealed, dirty)
	}
}

func TestStartNextExamEndedAndExportedAreIdempotent(t *testing.T) {
	t.Run("ended", func(t *testing.T) {
		a, store, _, _ := nextExamApp(t)
		if _, err := store.SealCurrent(); err != nil {
			t.Fatal(err)
		}
		before, _ := store.Snapshot()

		result, err := a.StartNextExam(true)
		if err != nil {
			t.Fatal(err)
		}
		after, dirty := store.Snapshot()
		if result.PreviousState != questionbank.Ended || result.SessionState != questionbank.Ended || result.Sealed || dirty {
			t.Fatalf("ended App 幂等结果错误: result=%+v after=%+v dirty=%v", result, after, dirty)
		}
		if before.EndedAt == nil || after.EndedAt == nil || !after.EndedAt.Equal(*before.EndedAt) {
			t.Fatal("ended App 幂等调用改写了 EndedAt")
		}
	})

	t.Run("exported", func(t *testing.T) {
		a, store, _, _ := nextExamApp(t)
		id, err := store.SealCurrent()
		if err != nil {
			t.Fatal(err)
		}
		exportResult, err := store.ExportSession(id, questionbank.ExportOptions{Name: "App 幂等导出"})
		if err != nil {
			t.Fatal(err)
		}
		before, _ := store.Snapshot()

		result, err := a.StartNextExam(true)
		if err != nil {
			t.Fatal(err)
		}
		after, dirty := store.Snapshot()
		if result.PreviousState != questionbank.Exported || result.SessionState != questionbank.Exported || result.Sealed || dirty {
			t.Fatalf("exported App 幂等结果错误: result=%+v after=%+v dirty=%v", result, after, dirty)
		}
		if after.ExportDirectory != exportResult.Directory || after.ExportedAt == nil || before.ExportedAt == nil || !after.ExportedAt.Equal(*before.ExportedAt) || !reflect.DeepEqual(after.Exports, before.Exports) {
			t.Fatalf("exported App 幂等调用破坏导出元数据: before=%+v after=%+v", before, after)
		}
	})
}

func TestStartNextExamRejectsDiscardedAndEmptyWithoutSideEffects(t *testing.T) {
	t.Run("discarded", func(t *testing.T) {
		a, store, _, recordID := nextExamApp(t)
		if err := store.DiscardCurrent(); err != nil {
			t.Fatal(err)
		}
		before, _ := store.Snapshot()

		result, err := a.StartNextExam(true)
		if err == nil || !strings.Contains(err.Error(), "已清空") {
			t.Fatalf("discarded 未拒绝: result=%+v err=%v", result, err)
		}
		assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Discarded)
	})

	t.Run("empty", func(t *testing.T) {
		a, _, _, recordID := nextExamApp(t)
		emptyStore, err := questionbank.NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		a.questionBank.store = emptyStore

		result, err := a.StartNextExam(true)
		if err == nil || !strings.Contains(err.Error(), "暂无记录") {
			t.Fatalf("empty 未拒绝: result=%+v err=%v", result, err)
		}
		if current, dirty := emptyStore.Snapshot(); current != nil || dirty {
			t.Fatalf("empty 拒绝后预建了分段: current=%+v dirty=%v", current, dirty)
		}
		if a.conversationEpoch != 7 || len(a.conversationRefs) != 1 || a.conversationRefs[0].RecordID != recordID || len(a.screenshotBuffer) != 2 || len(a.solver.SnapshotConversation().Turns) != 1 || a.questionBank.message != "原状态消息" {
			t.Fatalf("empty 拒绝后 App 状态变化: refs=%+v screenshots=%+v epoch=%d", a.conversationRefs, a.screenshotBuffer, a.conversationEpoch)
		}
	})
}

func TestStartNextExamPersistenceFailureHasZeroRuntimeSideEffects(t *testing.T) {
	a, store, _, recordID := nextExamApp(t)
	before, _ := store.Snapshot()
	manifest := filepath.Join(store.Root(), "_记录", before.ID, "session.json")
	backup := manifest + ".bak"
	if err := os.Rename(manifest, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(manifest, 0700); err != nil {
		t.Fatal(err)
	}

	result, err := a.StartNextExam(true)
	if err == nil {
		t.Fatalf("持久化失败未返回: %+v", result)
	}
	assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
}
