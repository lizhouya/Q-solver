package app

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/questionbank"
)

func TestDiscardQuestionBankSegmentPhysicallyDeletesThenClearsContext(t *testing.T) {
	a, store, _, _ := nextExamApp(t)
	before, _ := store.Snapshot()
	dir := store.Root() + string(os.PathSeparator) + "_记录" + string(os.PathSeparator) + before.ID
	bridge, err := mobile.NewBridge(9)
	if err != nil {
		t.Fatal(err)
	}
	bridge.Publish("solution-stream-start")
	bridge.Publish("solution-stream-chunk", "手机端保留内容")
	bridge.Publish("solution", "手机端保留内容")
	a.mobileBridge = bridge
	a.mobileRuntimeState = config.MobileRuntimeState{StableLink: true, Token: "synthetic-token", ClearSeq: 9, Generation: 2}
	a.mobileStateReady = true
	a.mobileStatus = mobileStatusReady
	mobileBefore := bridge.GetState()
	runtimeBefore := a.mobileRuntimeState

	if err = a.DiscardQuestionBankSegment(true); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("清空成功后分段目录仍存在: %v", err)
	}
	if current, dirty := store.Snapshot(); current != nil || dirty {
		t.Fatalf("清空成功后 Store 未释放 current: current=%+v dirty=%v", current, dirty)
	}
	if len(a.solver.SnapshotConversation().Turns) != 0 || len(a.conversationRefs) != 0 || a.conversationEpoch != 8 {
		t.Fatalf("清空成功后模型上下文未清理: refs=%+v epoch=%d conversation=%+v", a.conversationRefs, a.conversationEpoch, a.solver.SnapshotConversation())
	}
	if len(a.screenshotBuffer) != 2 {
		t.Fatalf("清空已完成分段不应删除尚未发送截图: %+v", a.screenshotBuffer)
	}
	if a.questionBank.message != "当前分段和截图已从本地永久删除" {
		t.Fatalf("成功消息不明确: %q", a.questionBank.message)
	}
	if a.mobileRuntimeState != runtimeBefore || !reflect.DeepEqual(bridge.GetState(), mobileBefore) || !a.mobileStateReady || a.mobileStatus != mobileStatusReady {
		t.Fatal("清空本地分段不应修改 Mobile 当前显示或固定链接")
	}
}

func TestDiscardQuestionBankSegmentRequiresConfirmationWithoutSideEffects(t *testing.T) {
	a, store, _, recordID := nextExamApp(t)
	before, _ := store.Snapshot()

	err := a.DiscardQuestionBankSegment(false)
	if err == nil || !strings.Contains(err.Error(), "永久删除") {
		t.Fatalf("未确认时未明确拒绝永久删除: %v", err)
	}
	assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
}

func TestDiscardQuestionBankSegmentRejectsInFlightAndExportingWithoutSideEffects(t *testing.T) {
	t.Run("cancelled but incomplete task", func(t *testing.T) {
		a, store, _, recordID := nextExamApp(t)
		before, _ := store.Snapshot()
		_, taskID := a.taskManager.StartTask("solve")
		if !a.taskManager.CancelCurrentTask() {
			t.Fatal("测试前置取消任务失败")
		}
		err := a.DiscardQuestionBankSegment(true)
		a.taskManager.CompleteTask(taskID)
		if err == nil || !strings.Contains(err.Error(), "完全结束") {
			t.Fatalf("未拒绝尚未完成的取消任务: %v", err)
		}
		assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
	})

	t.Run("exporting operation", func(t *testing.T) {
		a, store, _, recordID := nextExamApp(t)
		before, _ := store.Snapshot()
		a.questionBank.operation = "exporting"
		err := a.DiscardQuestionBankSegment(true)
		if err == nil || !strings.Contains(err.Error(), "整理或导出") {
			t.Fatalf("未拒绝导出状态: %v", err)
		}
		assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
	})
}

func TestDiscardQuestionBankSegmentStoreRejectionHasZeroRuntimeSideEffects(t *testing.T) {
	a, store, image, recordID := nextExamApp(t)
	before, _ := store.Snapshot()
	runningID, err := store.Begin([]string{image}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if runningID == "" {
		t.Fatal("测试前置未建立 running 记录")
	}

	err = a.DiscardQuestionBankSegment(true)
	if err == nil || !strings.Contains(err.Error(), "正在处理") {
		t.Fatalf("Store 拒绝未向上传递: %v", err)
	}
	assertNextExamRuntimeUnchanged(t, a, store, before.ID, recordID, questionbank.Active)
	current, _ := store.Snapshot()
	if len(current.Records) != 2 || current.Records[1].ID != runningID || current.Records[1].Status != questionbank.Running {
		t.Fatalf("拒绝后 running 记录变化: %+v", current.Records)
	}
}
