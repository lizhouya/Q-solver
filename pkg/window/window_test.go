package window

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"Q-Solver/pkg/platform"
)

// mockController 模拟平台窗口操作，验证管理器状态机而不依赖真实 Win32。
type mockController struct {
	mu         sync.Mutex
	showCalled bool
	hideCalled bool
	showErr    error
	hideErr    error
}

func (m *mockController) Show(hwnd platform.WindowHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.showCalled = true
	return m.showErr
}

func (m *mockController) Hide(hwnd platform.WindowHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hideCalled = true
	return m.hideErr
}

func (m *mockController) wasShowCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.showCalled
}

func (m *mockController) wasHideCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hideCalled
}

// reset 清零调用标志，用于幂等性测试。
func (m *mockController) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.showCalled = false
	m.hideCalled = false
}

const fakeHwnd = platform.WindowHandle(0x1234)

// newManager 构造测试管理器（注入 mock 控制器与 HWND provider）。
func newManager(c WindowController, provider func() platform.WindowHandle) *WindowVisibilityManager {
	return &WindowVisibilityManager{
		state:        WindowVisible,
		controller:   c,
		hwndProvider: provider,
	}
}

// --- Initial state = Visible ---

func TestInitialStateVisible(t *testing.T) {
	m := newManager(&mockController{}, func() platform.WindowHandle { return fakeHwnd })
	if m.State() != WindowVisible {
		t.Fatalf("初始状态应为 visible，得到 %v", m.State())
	}
	if !m.IsVisible() {
		t.Fatal("IsVisible() 初始应为 true")
	}
}

// --- Hide ---

func TestHide(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	if err := m.Hide(); err != nil {
		t.Fatalf("Hide 不应出错: %v", err)
	}
	if m.State() != WindowHidden {
		t.Fatalf("Hide 后状态应为 Hidden，得到 %v", m.State())
	}
	if !mock.wasHideCalled() {
		t.Fatal("Hide 应该调用平台控制器")
	}
	if mock.wasShowCalled() {
		t.Fatal("Hide 不应调用 show 控制器")
	}
}

// --- Show ---

func TestShow(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	if err := m.Hide(); err != nil {
		t.Fatalf("Hide 出错: %v", err)
	}
	if err := m.Show(); err != nil {
		t.Fatalf("Show 不应出错: %v", err)
	}
	if m.State() != WindowVisible {
		t.Fatalf("Show 后状态应为 Visible，得到 %v", m.State())
	}
	if !mock.wasShowCalled() {
		t.Fatal("Show 应该调用平台控制器")
	}
}

// --- Toggle 反转 ---

func TestToggle(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	// 初始 Visible → Toggle → Hidden
	st, err := m.Toggle()
	if err != nil {
		t.Fatalf("第一次 Toggle 出错: %v", err)
	}
	if st != WindowHidden {
		t.Fatalf("第一次 Toggle 后应为 Hidden，得到 %v", st)
	}
	if !mock.wasHideCalled() {
		t.Fatal("第一次 Toggle 应调用 Hide")
	}

	// Hidden → Toggle → Visible
	st, err = m.Toggle()
	if err != nil {
		t.Fatalf("第二次 Toggle 出错: %v", err)
	}
	if st != WindowVisible {
		t.Fatalf("第二次 Toggle 后应为 Visible，得到 %v", st)
	}
	if !mock.wasShowCalled() {
		t.Fatal("第二次 Toggle 应调用 Show")
	}
}

// --- 幂等性：连续 Hide / 初始 Show ---

func TestHideIdempotent(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	_ = m.Hide()
	mock.reset()
	if err := m.Hide(); err != nil {
		t.Fatalf("重复 Hide 不应出错: %v", err)
	}
	if mock.wasHideCalled() {
		t.Fatal("已隐藏时不应重复调用平台 Hide（幂等）")
	}
	if m.State() != WindowHidden {
		t.Fatal("重复 Hide 后状态应保持 Hidden")
	}
}

func TestShowIdempotent(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	if err := m.Show(); err != nil {
		t.Fatalf("初始可见时 Show 不应出错: %v", err)
	}
	if mock.wasShowCalled() {
		t.Fatal("初始可见时 Show 不应调用平台控制器（幂等）")
	}
	if m.State() != WindowVisible {
		t.Fatal("状态应保持 Visible")
	}
}

// --- 句柄不可用时的行为 ---

func TestHideNoHwnd(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return 0 })

	if err := m.Hide(); err == nil {
		t.Fatal("句柄不可用时 Hide 应返回错误")
	}
	if mock.wasHideCalled() {
		t.Fatal("句柄不可用时不应调用平台控制器")
	}
	if m.State() != WindowVisible {
		t.Fatal("句柄不可用时状态不应改变")
	}
}

func TestShowNoHwnd(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return 0 })
	m.state = WindowHidden // 模拟句柄丢失时已隐藏

	if err := m.Show(); err == nil {
		t.Fatal("句柄不可用时 Show 应返回错误")
	}
	if mock.wasShowCalled() {
		t.Fatal("句柄不可用时不应调用平台控制器")
	}
	if m.State() != WindowHidden {
		t.Fatal("句柄不可用时状态不应改变")
	}
}

// --- 控制器失败时，状态不应改变 ---

func TestHideControllerError(t *testing.T) {
	mock := &mockController{hideErr: errors.New("mock hide error")}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	if err := m.Hide(); err == nil {
		t.Fatal("控制器失败时 Hide 应返回错误")
	}
	if m.State() != WindowVisible {
		t.Fatal("控制器失败时状态不应改变")
	}
}

func TestShowControllerError(t *testing.T) {
	mock := &mockController{showErr: errors.New("mock show error")}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })
	m.state = WindowHidden

	if err := m.Show(); err == nil {
		t.Fatal("控制器失败时 Show 应返回错误")
	}
	if m.State() != WindowHidden {
		t.Fatal("控制器失败时状态不应改变")
	}
}

// --- 可见性回调 ---

func TestVisibilityCallback(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock, func() platform.WindowHandle { return fakeHwnd })

	var seen []WindowVisibleState
	m.SetVisibilityChange(func(s WindowVisibleState) {
		seen = append(seen, s)
	})

	_ = m.Hide()
	_ = m.Show()
	_ = m.Hide()

	want := []WindowVisibleState{WindowHidden, WindowVisible, WindowHidden}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("回调序列 = %v, 期望 %v", seen, want)
	}
}

// --- State.String ---

func TestStateString(t *testing.T) {
	if WindowVisible.String() != "visible" {
		t.Fatalf("visible 应映射为 'visible'，得到 %s", WindowVisible.String())
	}
	if WindowHidden.String() != "hidden" {
		t.Fatalf("hidden 应映射为 'hidden'，得到 %s", WindowHidden.String())
	}
}