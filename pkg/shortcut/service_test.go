package shortcut

import (
	"errors"
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
	clickThroughState       bool
	clickThroughErr         error
	toggleWindowCalls       int
	moveCalls               []string
	scrollCalls             []string
	events                  []string
	eventData               [][]interface{}
}

func (m *mockDelegate) TriggerSolve()            { m.solveCalls++ }
func (m *mockDelegate) TriggerScreenshot()       { m.screenshotCalls++ }
func (m *mockDelegate) TriggerSend()             { m.sendCalls++ }
func (m *mockDelegate) TriggerDeleteScreenshot() { m.deleteScreenshotCalls++ }
func (m *mockDelegate) ToggleVisibility()        { m.toggleVisibilityCalls++ }
func (m *mockDelegate) ToggleClickThrough() (bool, error) {
	m.toggleClickThroughCalls++
	return m.clickThroughState, m.clickThroughErr
}
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
	m.eventData = append(m.eventData, data)
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

func TestHandleRecordOnlyPreviewsBinding(t *testing.T) {
	d := &mockDelegate{}
	initial := map[string]KeyBinding{
		"toggle_window": {ComboID: "72+160+162", KeyName: "Ctrl+Shift+H"},
	}
	s := NewService(d, initial, noopSubscribe)

	s.handleRecord("toggle_window", "Shift+C", "67+160")

	if got := s.GetShortcuts()["toggle_window"]; got != initial["toggle_window"] {
		t.Fatalf("录制预览不得提前修改运行态绑定：want %+v, got %+v", initial["toggle_window"], got)
	}
	if !reflect.DeepEqual(d.events, []string{"key-recorded"}) {
		t.Fatalf("预览应只推送 key-recorded，实际 %v", d.events)
	}
}

func TestHandleRecordingCompleteConflictKeepsBothOriginalBindings(t *testing.T) {
	d := &mockDelegate{}
	initial := map[string]KeyBinding{
		"toggle_window": {ComboID: "72+160+162", KeyName: "Ctrl+Shift+H"},
		"screenshot":    {ComboID: "119", KeyName: "F8"},
	}
	s := NewService(d, initial, noopSubscribe)

	// 把 screenshot 录成 Ctrl+Shift+H → 与 toggle_window 冲突；预览和完成都不得污染运行态。
	s.handleRecord("screenshot", "Ctrl+Shift+H", "72+160+162")
	s.handleRecordingComplete("screenshot", "Ctrl+Shift+H", "72+160+162")

	if !reflect.DeepEqual(d.events, []string{"key-recorded", "shortcut-error"}) {
		t.Fatalf("冲突录制事件不正确：%v", d.events)
	}
	for action, want := range initial {
		if got := s.GetShortcuts()[action]; got != want {
			t.Fatalf("冲突后 %s 绑定被破坏：want %+v, got %+v", action, want, got)
		}
	}
}

func TestHandleRecordingCompleteCommitsAfterConflictCheck(t *testing.T) {
	d := &mockDelegate{}
	s := NewService(d, map[string]KeyBinding{
		"screenshot": {ComboID: "119", KeyName: "F8"},
	}, noopSubscribe)

	s.handleRecord("screenshot", "Shift+C", "67+160")
	s.handleRecordingComplete("screenshot", "Shift+C", "67+160")

	want := KeyBinding{ComboID: "67+160", KeyName: "Shift+C"}
	if got := s.GetShortcuts()["screenshot"]; got != want {
		t.Fatalf("无冲突完成后应提交绑定：want %+v, got %+v", want, got)
	}
	if !reflect.DeepEqual(d.events, []string{"key-recorded", "shortcut-saved"}) {
		t.Fatalf("成功录制应结束前端录制状态，实际事件 %v", d.events)
	}
}

func assertSingleToast(t *testing.T, d *mockDelegate, wantText, wantType string) {
	t.Helper()
	if !reflect.DeepEqual(d.events, []string{"toast"}) {
		t.Fatalf("应只发送一次 toast，实际事件 %v", d.events)
	}
	if len(d.eventData) != 1 || len(d.eventData[0]) != 1 {
		t.Fatalf("toast payload 结构不正确：%v", d.eventData)
	}
	payload, ok := d.eventData[0][0].(map[string]string)
	if !ok {
		t.Fatalf("toast payload 应为 map[string]string，实际 %T", d.eventData[0][0])
	}
	if got := payload["text"]; got != wantText {
		t.Fatalf("toast 文案应为 %q，实际 %q", wantText, got)
	}
	if got := payload["type"]; got != wantType {
		t.Fatalf("toast type 应为 %q，实际 %q", wantType, got)
	}
	if len(payload) != 2 {
		t.Fatalf("toast payload 应仅包含 text/type，实际 %v", payload)
	}
}

func TestClickThroughEnabledToastUsesReturnedState(t *testing.T) {
	d := &mockDelegate{clickThroughState: true}
	s := newTestService(d)

	s.handleTrigger("clickthrough")

	if d.toggleClickThroughCalls != 1 {
		t.Fatalf("clickthrough 应调用 ToggleClickThrough 1 次，实际 %d", d.toggleClickThroughCalls)
	}
	assertSingleToast(t, d, "鼠标穿透：已开启", "success")
}

func TestClickThroughDisabledToastUsesReturnedState(t *testing.T) {
	d := &mockDelegate{clickThroughState: false}
	s := newTestService(d)

	s.handleTrigger("clickthrough")

	if d.toggleClickThroughCalls != 1 {
		t.Fatalf("clickthrough 应调用 ToggleClickThrough 1 次，实际 %d", d.toggleClickThroughCalls)
	}
	assertSingleToast(t, d, "鼠标穿透：已关闭", "info")
}

func TestClickThroughFailureDoesNotEmitSuccessToast(t *testing.T) {
	d := &mockDelegate{
		clickThroughState: true,
		clickThroughErr:   errors.New("platform failure"),
	}
	s := newTestService(d)

	s.handleTrigger("clickthrough")

	if d.toggleClickThroughCalls != 1 {
		t.Fatalf("失败时 clickthrough 仍应只调用 ToggleClickThrough 1 次，实际 %d", d.toggleClickThroughCalls)
	}
	assertSingleToast(t, d, "鼠标穿透切换失败", "error")
}

func TestClickThroughCustomBindingStillUsesActionState(t *testing.T) {
	d := &mockDelegate{clickThroughState: true}
	initial := map[string]KeyBinding{
		"clickthrough": {ComboID: "75+162", KeyName: "Ctrl+K"},
	}
	s := NewService(d, initial, noopSubscribe)

	s.handleTrigger("clickthrough")

	if got := s.GetShortcuts()["clickthrough"].KeyName; got != "Ctrl+K" {
		t.Fatalf("测试前提：自定义绑定应为 Ctrl+K，实际 %s", got)
	}
	if d.toggleClickThroughCalls != 1 {
		t.Fatalf("自定义快捷键下应调用 ToggleClickThrough 1 次，实际 %d", d.toggleClickThroughCalls)
	}
	assertSingleToast(t, d, "鼠标穿透：已开启", "success")
}

func TestHandleTriggerStartNextExamOnlyEmitsRequest(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	s.handleTrigger("start_next_exam")

	if !reflect.DeepEqual(d.events, []string{"request-start-next-exam"}) {
		t.Fatalf("开始下一套快捷键应只发一次请求事件，实际 %v", d.events)
	}
	if len(d.eventData) != 1 || len(d.eventData[0]) != 0 {
		t.Fatalf("开始下一套请求事件不得携带 payload，实际 %v", d.eventData)
	}
	if d.solveCalls != 0 || d.screenshotCalls != 0 || d.sendCalls != 0 || d.deleteScreenshotCalls != 0 ||
		d.toggleVisibilityCalls != 0 || d.toggleClickThroughCalls != 0 ||
		d.toggleWindowCalls != 0 || len(d.moveCalls) != 0 || len(d.scrollCalls) != 0 {
		t.Fatal("开始下一套请求不得直接调用任何 mutation 或其他快捷键 delegate")
	}
}

func TestHandleTriggerClearMobileDisplayOnlyRequestsConfirmation(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)

	s.handleTrigger("clear_mobile_display")

	if !reflect.DeepEqual(d.events, []string{"request-clear-mobile-display"}) {
		t.Fatalf("清屏快捷键应只发一次请求事件，实际 %v", d.events)
	}
	if len(d.eventData) != 1 || len(d.eventData[0]) != 0 {
		t.Fatalf("清屏请求事件不得携带 payload，实际 %v", d.eventData)
	}
	if d.screenshotCalls != 0 || d.sendCalls != 0 || d.deleteScreenshotCalls != 0 ||
		d.toggleVisibilityCalls != 0 || d.toggleClickThroughCalls != 0 ||
		d.toggleWindowCalls != 0 || len(d.moveCalls) != 0 || len(d.scrollCalls) != 0 {
		t.Fatal("清屏请求不得触发其他快捷键 delegate")
	}
}
