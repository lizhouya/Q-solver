package shortcut

import (
	"reflect"
	"testing"
)

// mockDelegate 实现 ServiceDelegate，记录各触发方法的调用次数与事件。
type mockDelegate struct {
	solveCalls              int
	screenshotCalls         int
	sendCalls               int
	deleteScreenshotCalls   int
	toggleVisibilityCalls   int
	toggleClickThroughCalls int
	toggleWindowCalls       int
	moveCalls               []string
	scrollCalls             []string
	events                  []string
}

func (m *mockDelegate) TriggerSolve()            { m.solveCalls++ }
func (m *mockDelegate) TriggerScreenshot()       { m.screenshotCalls++ }
func (m *mockDelegate) TriggerSend()             { m.sendCalls++ }
func (m *mockDelegate) TriggerDeleteScreenshot() { m.deleteScreenshotCalls++ }
func (m *mockDelegate) ToggleVisibility()        { m.toggleVisibilityCalls++ }
func (m *mockDelegate) ToggleClickThrough()      { m.toggleClickThroughCalls++ }
func (m *mockDelegate) ToggleWindowVisible() (string, error) {
	m.toggleWindowCalls++
	return "hidden", nil
}
func (m *mockDelegate) MoveWindow(dx, dy int) { m.moveCalls = append(m.moveCalls, "move") }
func (m *mockDelegate) ScrollContent(direction string) {
	m.scrollCalls = append(m.scrollCalls, direction)
}
func (m *mockDelegate) EmitEvent(eventName string, data ...interface{}) {
	m.events = append(m.events, eventName)
}

// noopSubscribe 空配置订阅回调，保证测试不依赖 ConfigManager。
func noopSubscribe(func(map[string]KeyBinding)) {}

// newTestService 构造一个隔离的 Service（不启动 Windows 钩子）。
func newTestService(d *mockDelegate) *Service {
	return NewService(d, map[string]KeyBinding{}, noopSubscribe)
}

func TestHandleTriggerToggleWindow(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	s.handleTrigger("toggle_window")

	if d.toggleWindowCalls != 1 {
		t.Fatalf("toggle_window 应触发 delegate.ToggleWindowVisible 1 次，实际 %d", d.toggleWindowCalls)
	}
	// ToggleWindowVisible 的 error 仅在失败时记日志，不 panic（此处确认无副作用污染其它 action）
	if d.toggleVisibilityCalls != 0 {
		t.Fatalf("toggle_window 不得触发 F9 隐身 toggle，实际 %d", d.toggleVisibilityCalls)
	}
}

func TestToggleWindowDistinctFromToggle(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	s.handleTrigger("toggle_window")
	s.handleTrigger("toggle")

	if d.toggleWindowCalls != 1 {
		t.Fatalf("toggle_window 应独立计数 1，实际 %d", d.toggleWindowCalls)
	}
	if d.toggleVisibilityCalls != 1 {
		t.Fatalf("toggle（F9 防录屏隐身）应独立计数 1，实际 %d", d.toggleVisibilityCalls)
	}
}

func TestHandleTriggerExistingActionsUnaffected(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	// 既有 action 均不受 toggle_window 影响
	s.handleTrigger("screenshot")
	s.handleTrigger("send")
	s.handleTrigger("delete")
	s.handleTrigger("toggle")
	s.handleTrigger("clickthrough")

	if d.toggleWindowCalls != 0 {
		t.Fatalf("既有 action 不应触发 toggle_window，实际 %d 次", d.toggleWindowCalls)
	}
	if d.screenshotCalls != 1 {
		t.Fatalf("screenshot 应触发 1 次，实际 %d", d.screenshotCalls)
	}
	if d.sendCalls != 1 {
		t.Fatalf("send 应触发 1 次，实际 %d", d.sendCalls)
	}
	if d.deleteScreenshotCalls != 1 {
		t.Fatalf("delete 应触发 1 次，实际 %d", d.deleteScreenshotCalls)
	}
	if d.toggleVisibilityCalls != 1 {
		t.Fatalf("toggle 应触发 1 次，实际 %d", d.toggleVisibilityCalls)
	}
	if d.toggleClickThroughCalls != 1 {
		t.Fatalf("clickthrough 应触发 1 次，实际 %d", d.toggleClickThroughCalls)
	}
}

func TestCtrlShiftHComboMapping(t *testing.T) {
	// Ctrl(162 VK_LCONTROL) + Shift(160 VK_LSHIFT) + H(72) → ComboID 升序
	keys := map[uint32]bool{162: true, 160: true, 72: true}
	if got := GetComboID(keys); got != "72+160+162" {
		t.Fatalf("ComboID 应为 72+160+162，实际 %s", got)
	}
	if got := GetReadableName(keys); got != "Ctrl+Shift+H" {
		t.Fatalf("可读名应为 Ctrl+Shift+H，实际 %s", got)
	}
}

func TestHandleRecordStoresToggleWindow(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	s.handleRecord("toggle_window", "Ctrl+Shift+H", "72+160+162")

	bound, ok := s.GetShortcuts()["toggle_window"]
	if !ok {
		t.Fatal("录制后应保存 toggle_window 绑定")
	}
	if bound.ComboID != "72+160+162" || bound.KeyName != "Ctrl+Shift+H" {
		t.Fatalf("绑定不对：%+v", bound)
	}
	if !reflect.DeepEqual(d.events, []string{"key-recorded"}) {
		t.Fatalf("应推送 key-recorded 事件，实际 %v", d.events)
	}
}

func TestHandleRecordingCompleteConflict(t *testing.T) {
	d := &mockDelegate{}
	initial := map[string]KeyBinding{
		"toggle_window": {ComboID: "72+160+162", KeyName: "Ctrl+Shift+H"},
		"screenshot":    {ComboID: "119", KeyName: "F8"},
	}
	s := NewService(d, initial, noopSubscribe)

	// 把 screenshot 录成 Ctrl+Shift+H → 与 toggle_window 冲突，应报错且不破坏原绑定
	s.handleRecordingComplete("screenshot", "Ctrl+Shift+H", "72+160+162")

	found := false
	for _, e := range d.events {
		if e == "shortcut-error" {
			found = true
		}
	}
	if !found {
		t.Fatal("冲突时应推送 shortcut-error 事件")
	}
	if got := s.GetShortcuts()["toggle_window"].ComboID; got != "72+160+162" {
		t.Fatalf("冲突后 toggle_window 绑定被破坏：%s", got)
	}
}
