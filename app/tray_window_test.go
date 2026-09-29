package app

import (
	"sync"
	"testing"

	"Q-Solver/pkg/platform"
	"Q-Solver/pkg/window"
)

// ==================== Phase 3.1 一致性收口测试 ====================
// 验证 app 层全部「Hidden ↔ Visible」业务入口都经由 WindowVisibilityManager 状态机：
//   - trayOnHide → windowVisibility.Hide()（恰好一次）
//   - app.Show()   → 状态从 Hidden 恢复 Visible，且已可见时幂等（不重复调用平台）

// fakeWindowController 记录平台 Show/Hide 调用次数，代替真机 Win32。
type fakeWindowController struct {
	mu        sync.Mutex
	showCalls int
	hideCalls int
}

func (c *fakeWindowController) Show(platform.WindowHandle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.showCalls++
	return nil
}

func (c *fakeWindowController) Hide(platform.WindowHandle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hideCalls++
	return nil
}

func (c *fakeWindowController) calls() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.showCalls, c.hideCalls
}

// newVisibilityApp 构造带注入控制器 WindowVisibilityManager 的 App 测试实例。
// hwndProvider 恒返回非零假句柄（控制器为 mock，不触碰真实 Win32）。
func newVisibilityApp(fc *fakeWindowController) *App {
	a := &App{
		windowVisibility: window.NewWindowVisibilityManagerWithController(
			func() platform.WindowHandle { return platform.WindowHandle(0x1234) },
			fc,
		),
	}
	return a
}

// --- Tray Hide：trayOnHide → visibility.Hide() 恰好一次 ---

func TestTrayHideCallsVisibilityHideOnce(t *testing.T) {
	fc := &fakeWindowController{}
	a := newVisibilityApp(fc)

	a.trayOnHide()

	if a.windowVisibility.IsVisible() {
		t.Fatal("trayOnHide 后窗口应处于 Hidden")
	}
	shows, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("trayOnHide 应恰好调用 Hide 1 次，实际 %d", hides)
	}
	if shows != 0 {
		t.Fatalf("Hide 路径不应调用平台 Show，实际 %d", shows)
	}
}

// 已隐藏时重复 trayOnHide → 幂等，不重复调用平台 Hide。
func TestTrayHideIdempotent(t *testing.T) {
	fc := &fakeWindowController{}
	a := newVisibilityApp(fc)

	a.trayOnHide()
	a.trayOnHide()

	_, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("重复 trayOnHide 应只调用 Hide 1 次（幂等），实际 %d", hides)
	}
}

// --- App.Show：Hidden → Visible，且已可见时幂等 ---

func TestAppShowRestoresVisibleFromHidden(t *testing.T) {
	fc := &fakeWindowController{}
	a := newVisibilityApp(fc)

	// 先经状态机隐藏，再通过 app.Show()（SecondInstance/托盘统一入口）恢复
	if err := a.windowVisibility.Hide(); err != nil {
		t.Fatalf("预置 Hide 失败: %v", err)
	}

	a.Show()

	if !a.windowVisibility.IsVisible() {
		t.Fatal("app.Show 后窗口应为 Visible（状态机已同步）")
	}
	shows, hides := fc.calls()
	if shows != 1 {
		t.Fatalf("Show 应调用平台 Show 1 次（含预置 Hide 后恢复），实际 %d", shows)
	}
	if hides != 1 {
		t.Fatalf("预置 Hide 应调用 1 次，实际 %d", hides)
	}
}

func TestAppShowIdempotentWhenVisible(t *testing.T) {
	fc := &fakeWindowController{}
	a := newVisibilityApp(fc)

	// 初始即 Visible → 重复 Show 应幂等
	a.Show()
	a.Show()

	shows, hides := fc.calls()
	if shows != 0 {
		t.Fatalf("初始可见时 Show 不应调用平台 Show（幂等），实际 %d", shows)
	}
	if hides != 0 {
		t.Fatalf("Show 路径不应调用平台 Hide，实际 %d", hides)
	}
	if !a.windowVisibility.IsVisible() {
		t.Fatal("状态应保持 Visible")
	}
}
