package mobile

import "testing"

func TestScreenshotStateIsReplayableIdempotentAndSurvivesReset(t *testing.T) {
	b := newTestBridge(t)
	b.SetHints([]ShortcutHint{
		{Label: "截图", KeyName: "F8"},
		{Label: "发送解题", KeyName: "Ctrl+J"},
		{Label: "删除截图", KeyName: "Ctrl+D"},
	})
	sub, _ := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.SetScreenshotState(2, 3)
	updated := <-sub
	if updated.ScreenshotCount != 2 || updated.ScreenshotLimit != 3 {
		t.Fatalf("截图状态错误: %+v", updated)
	}
	if updated.Payload.Answer != "" || len(updated.Hints) != 3 {
		t.Fatalf("截图元数据更新不应污染答案或快捷键: %+v", updated)
	}
	if replay := b.GetState(); replay.ScreenshotCount != 2 || replay.ScreenshotLimit != 3 {
		t.Fatalf("截图状态不可重放: %+v", replay)
	}

	b.SetScreenshotState(2, 3)
	if len(sub) != 0 {
		t.Fatal("相同截图状态不应重复广播")
	}

	beforeReset := b.GetState()
	for len(sub) < cap(sub) {
		b.Publish("solution-stream-chunk", "queued")
	}
	b.Reset()
	if len(sub) != 1 {
		t.Fatalf("Reset 必须排空积压并只保留 idle 屏障，队列长度 %d", len(sub))
	}
	reset := <-sub
	if reset.Status != StatusIdle || reset.Seq <= beforeReset.Seq {
		t.Fatalf("Reset 未发布新 idle 边界: before=%+v after=%+v", beforeReset, reset)
	}
	if reset.ClearSeq != beforeReset.ClearSeq || reset.ScreenshotCount != 2 || reset.ScreenshotLimit != 3 || len(reset.Hints) != 3 {
		t.Fatalf("Reset 未保留 clearSeq/截图状态/快捷键: %+v", reset)
	}

	b.ClearDisplay(reset.ClearSeq + 1)
	barrier := <-sub
	if barrier.ScreenshotCount != 2 || barrier.ScreenshotLimit != 3 {
		t.Fatalf("ClearDisplay 不应丢失截图状态: %+v", barrier)
	}

	b.SetScreenshotState(-1, -2)
	normalized := <-sub
	if normalized.ScreenshotCount != 0 || normalized.ScreenshotLimit != 0 {
		t.Fatalf("负数截图状态应归零: %+v", normalized)
	}
	b.SetScreenshotState(9, 3)
	clamped := <-sub
	if clamped.ScreenshotCount != 3 || clamped.ScreenshotLimit != 3 {
		t.Fatalf("截图数量应限制在上限内: %+v", clamped)
	}
}
