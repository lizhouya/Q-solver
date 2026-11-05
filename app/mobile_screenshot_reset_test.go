package app

import (
	"reflect"
	"testing"

	"Q-Solver/pkg/mobile"
)

func bridgeWithCompletedAnswer(t *testing.T, count int) *mobile.Bridge {
	t.Helper()
	bridge, err := mobile.NewBridge(17)
	if err != nil {
		t.Fatal(err)
	}
	bridge.SetHints([]mobile.ShortcutHint{
		{Label: "截图", KeyName: "F8"},
		{Label: "发送解题", KeyName: "Ctrl+J"},
		{Label: "删除截图", KeyName: "Ctrl+D"},
	})
	bridge.SetScreenshotState(count, MaxScreenshots)
	bridge.Publish("solution-stream-start")
	bridge.Publish("solution-stream-chunk", "同步端旧答案")
	bridge.Publish("solution", "同步端旧答案")
	return bridge
}

func TestScreenshotBufferRemovalAndClearSyncMobileCount(t *testing.T) {
	bridge := bridgeWithCompletedAnswer(t, 2)
	a := &App{mobileBridge: bridge, screenshotBuffer: []string{"one", "two"}}

	a.RemoveLastScreenshot()
	if state := bridge.GetState(); state.ScreenshotCount != 1 || state.ScreenshotLimit != MaxScreenshots || state.Status != mobile.StatusDone {
		t.Fatalf("删除截图后 Mobile 状态错误: %+v", state)
	}

	a.ClearScreenshots()
	if state := bridge.GetState(); state.ScreenshotCount != 0 || state.ScreenshotLimit != MaxScreenshots || state.Status != mobile.StatusDone {
		t.Fatalf("单独清截图只能归零数量，不应删除答案: %+v", state)
	}
}

func TestSplitQuestionBankSegmentResetsMobileBusinessButKeepsScreenshotState(t *testing.T) {
	a, _, _, _ := nextExamApp(t)
	bridge := bridgeWithCompletedAnswer(t, len(a.screenshotBuffer))
	a.mobileBridge = bridge
	before := bridge.GetState()

	if err := a.SplitQuestionBankSegment(); err != nil {
		t.Fatal(err)
	}
	after := bridge.GetState()
	if after.Status != mobile.StatusIdle || after.Seq <= before.Seq {
		t.Fatalf("结束本段后同步端未进入新 idle 边界: before=%+v after=%+v", before, after)
	}
	if after.ClearSeq != before.ClearSeq || after.ScreenshotCount != 2 || after.ScreenshotLimit != MaxScreenshots || !reflect.DeepEqual(after.Hints, before.Hints) {
		t.Fatalf("结束本段不应改变 clearSeq/截图状态/快捷键: before=%+v after=%+v", before, after)
	}
}

func TestRejectedLocalCleanupDoesNotResetMobileBusiness(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*App)
		invoke  func(*App) error
	}{
		{
			name:    "split unavailable",
			prepare: func(a *App) { a.questionBank.store = nil },
			invoke:  func(a *App) error { return a.SplitQuestionBankSegment() },
		},
		{
			name:   "discard unconfirmed",
			invoke: func(a *App) error { return a.DiscardQuestionBankSegment(false) },
		},
		{
			name: "next exam unconfirmed",
			invoke: func(a *App) error {
				_, err := a.StartNextExam(false)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, _, _, _ := nextExamApp(t)
			bridge := bridgeWithCompletedAnswer(t, len(a.screenshotBuffer))
			a.mobileBridge = bridge
			if test.prepare != nil {
				test.prepare(a)
			}
			before := bridge.GetState()
			if err := test.invoke(a); err == nil {
				t.Fatal("测试前提：本地清理应被拒绝")
			}
			if after := bridge.GetState(); !reflect.DeepEqual(after, before) {
				t.Fatalf("本地清理失败不得重置 Mobile: before=%+v after=%+v", before, after)
			}
		})
	}
}
