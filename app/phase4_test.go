package app

import (
	"sync"
	"testing"

	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/platform"
	"Q-Solver/pkg/shortcut"
	"Q-Solver/pkg/window"
)

// Phase 4 自动隐藏测试：applyOutputModeWindowPolicy 的 transition-triggered 语义。
//
// 核心规则：
//   - 仅在 非 mobile → mobile 时触发 Hide exactly once（one-shot）
//   - mobile → desktop/dual 不自动 Show
//   - 用户在 mobile 模式手动 Show 后不再次强制 Hide
//   - Phase 4.1：仅当 Mobile Server 可用时自动 Hide

// fakeHideController 记录平台 Show/Hide 调用次数，与 Phase 3.1 相同模式。
type fakeHideController struct {
	mu        sync.Mutex
	showCalls int
	hideCalls int
}

func (c *fakeHideController) Show(platform.WindowHandle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.showCalls++
	return nil
}

func (c *fakeHideController) Hide(platform.WindowHandle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hideCalls++
	return nil
}

func (c *fakeHideController) calls() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.showCalls, c.hideCalls
}

// fakeDelegate 是 shortcut.ServiceDelegate 的最小实现，满足 NewService 编译。
type fakeDelegate struct{}

func (f *fakeDelegate) TriggerSolve()           {}
func (f *fakeDelegate) TriggerScreenshot()      {}
func (f *fakeDelegate) TriggerSend()            {}
func (f *fakeDelegate) TriggerDeleteScreenshot() {}
func (f *fakeDelegate) ToggleVisibility()        {}
func (f *fakeDelegate) ToggleClickThrough() (bool, error) { return false, nil }
func (f *fakeDelegate) MoveWindow(int, int)      {}
func (f *fakeDelegate) ScrollContent(string)     {}
func (f *fakeDelegate) ToggleWindowVisible() (string, error) { return "visible", nil }
func (f *fakeDelegate) EmitEvent(string, ...interface{})      {}

// startTestMobileServer 创建并启动测试用 Mobile Server（HTTP listener 绑定随机端口）。
// 用于需要 mobileOutputEnabled() 返回 true 的测试。
// 如果 port 38511 冲突，自动 skip 测试。
func startTestMobileServer(t *testing.T) (*mobile.Bridge, *mobile.Server) {
	t.Helper()
	bridge := mobile.NewBridge()
	srv, err := mobile.NewServer(bridge)
	if err != nil {
		t.Skipf("创建 mobile server 失败: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Skipf("启动 mobile server 失败（port 38511 可能冲突）: %v", err)
	}
	t.Cleanup(func() { srv.Stop() })
	return bridge, srv
}

// newPhase4App 构造带注入控制器的 App 测试实例，shortcutService 为真实 Service（非 nil）。
// mobileBridge / mobileServer 均为 nil → mobileOutputEnabled() 返回 false。
func newPhase4App(fc *fakeHideController) *App {
	return &App{
		windowVisibility: window.NewWindowVisibilityManagerWithController(
			func() platform.WindowHandle { return platform.WindowHandle(0x1234) },
			fc,
		),
		shortcutService: shortcut.NewService(
			&fakeDelegate{},
			map[string]shortcut.KeyBinding{},
			func(callback func(map[string]shortcut.KeyBinding)) {},
		),
	}
}

// newPhase4AppWithMobile 构造 App，同时装配 mobileBridge + mobileServer（已启动）。
// ✅ mobileOutputEnabled() 返回 true。
func newPhase4AppWithMobile(t *testing.T, fc *fakeHideController) *App {
	t.Helper()
	bridge, srv := startTestMobileServer(t)
	a := newPhase4App(fc)
	a.mobileBridge = bridge
	a.mobileServer = srv
	return a
}

// --- T-P4-01: Startup desktop → 不调用 Hide ---

func TestP4StartupDesktopNoHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc)

	a.applyOutputModeWindowPolicy("desktop", "desktop")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("desktop→desktop 不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-02: Startup dual → 不调用 Hide ---

func TestP4StartupDualNoHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc)

	a.applyOutputModeWindowPolicy("dual", "dual")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("dual→dual 不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-03: Startup mobile + server available → Hide once ---

func TestP4StartupMobileHides(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	a.applyOutputModeWindowPolicy("desktop", "mobile")

	shows, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("desktop→mobile 应 Hide 恰好 1 次，实际 %d", hides)
	}
	if shows != 0 {
		t.Fatalf("自动隐藏不应调用 Show，实际 %d", shows)
	}
	if a.windowVisibility.IsVisible() {
		t.Fatal("Hide 后窗口应处于 Hidden")
	}
}

// --- T-P4-04: Runtime desktop → mobile + server available → Hide once ---

func TestP4RuntimeDesktopToMobileHides(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	a.applyOutputModeWindowPolicy("desktop", "mobile")

	_, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("desktop→mobile 应 Hide 恰好 1 次，实际 %d", hides)
	}
}

// --- T-P4-05: Runtime dual → mobile + server available → Hide once ---

func TestP4RuntimeDualToMobileHides(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	a.applyOutputModeWindowPolicy("dual", "mobile")

	_, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("dual→mobile 应 Hide 恰好 1 次，实际 %d", hides)
	}
}

// --- T-P4-06: mobile → mobile → 不重复 Hide（one-shot） ---

func TestP4MobileToMobileNoRepeat(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	// 第一次进入 mobile
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	shows1, hides1 := fc.calls()
	if hides1 != 1 {
		t.Fatalf("首次进入 mobile 应 Hide 1 次，实际 %d", hides1)
	}
	_ = shows1

	// 第二次仍 mobile → 不应重复 Hide
	a.applyOutputModeWindowPolicy("mobile", "mobile")
	_, hides2 := fc.calls()
	if hides2 != 1 {
		t.Fatalf("mobile→mobile 不应重复 Hide，实际 %d", hides2)
	}
}

// --- T-P4-07: mobile → desktop → 不自动 Show ---

func TestP4MobileToDesktopNoAutoShow(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	// 先进入 mobile
	a.applyOutputModeWindowPolicy("desktop", "mobile")

	// 切回 desktop
	a.applyOutputModeWindowPolicy("mobile", "desktop")

	shows, hides := fc.calls()
	if shows != 0 {
		t.Fatalf("mobile→desktop 不应自动 Show，实际 %d", shows)
	}
	if hides != 1 {
		t.Fatalf("此前 mobile 的 Hide 应仍为 1 次，实际 %d", hides)
	}
}

// --- T-P4-08: mobile → dual → 不自动 Show ---

func TestP4MobileToDualNoAutoShow(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	a.applyOutputModeWindowPolicy("desktop", "mobile")
	a.applyOutputModeWindowPolicy("mobile", "dual")

	shows, hides := fc.calls()
	if shows != 0 {
		t.Fatalf("mobile→dual 不应自动 Show，实际 %d", shows)
	}
	if hides != 1 {
		t.Fatalf("此前 mobile 的 Hide 应仍为 1 次，实际 %d", hides)
	}
}

// --- T-P4-09: mobile 模式手动 Show 后不再次强制 Hide ---
//
// 模拟用户场景：
//   1. mode=mobile + server available → auto Hide（1 次）
//   2. 用户手动 Show（Ctrl+Shift+H / Tray）
//   3. mode 仍为 mobile → 不应再次 Hide
func TestP4MobileManualShowNoForcedHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4AppWithMobile(t, fc)

	// 1. auto hide
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	_, hides := fc.calls()
	if hides != 1 {
		t.Fatalf("初始 mobile 应 Hide 1 次，实际 %d", hides)
	}

	// 2. 用户手动 Show
	if err := a.windowVisibility.Show(); err != nil {
		t.Fatalf("手动 Show 失败: %v", err)
	}
	shows, hides := fc.calls()
	if shows != 1 {
		t.Fatalf("手动 Show 应调用 Show 1 次，实际 %d", shows)
	}
	if hides != 1 {
		t.Fatalf("手动 Show 后 Hide 计数不应增长，实际 %d", hides)
	}

	// 3. mode 仍为 mobile → 不应再次 Hide（one-shot 语义）
	a.applyOutputModeWindowPolicy("mobile", "mobile")
	shows2, hides2 := fc.calls()
	if hides2 != 1 {
		t.Fatalf("mobile 模式仍为 mobile 不应重复 Hide，实际 %d", hides2)
	}
	if shows2 != 1 {
		t.Fatalf("不应额外调用 Show，实际 %d", shows2)
	}

	if !a.windowVisibility.IsVisible() {
		t.Fatal("手动 Show 后窗口应保持 Visible（不应被强制隐藏）")
	}
}

// --- T-P4-10: Hide 失败不 panic ---
//
// 控制器 Hide 方法返回错误，验证 applyOutputModeWindowPolicy 不会 panic。

type errorHideController struct {
	mu        sync.Mutex
	hideCalls int
}

func (c *errorHideController) Show(platform.WindowHandle) error { return nil }

func (c *errorHideController) Hide(platform.WindowHandle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hideCalls++
	return errHideFailed
}

var errHideFailed = &hideError{}

type hideError struct{}

func (e *hideError) Error() string { return "模拟 Hide 失败" }

func TestP4HideFailureNoPanic(t *testing.T) {
	ehc := &errorHideController{}
	a := &App{
		windowVisibility: window.NewWindowVisibilityManagerWithController(
			func() platform.WindowHandle { return platform.WindowHandle(0x1234) },
			ehc,
		),
		shortcutService: shortcut.NewService(
			&fakeDelegate{},
			map[string]shortcut.KeyBinding{},
			func(callback func(map[string]shortcut.KeyBinding)) {},
		),
	}
	// 需要 mobile server 可用才能到达 Hide 调用
	bridge, srv := startTestMobileServer(t)
	a.mobileBridge = bridge
	a.mobileServer = srv

	// 不应 panic
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	// 验证调用了 Hide（即使返回错误）
	if ehc.hideCalls != 1 {
		t.Fatalf("应调用 Hide 1 次（即使失败），实际 %d", ehc.hideCalls)
	}
}

// --- T-P4-11: shortcutService 为 nil 时不自动 Hide（无恢复路径） ---

func TestP4NoShortcutNoHide(t *testing.T) {
	fc := &fakeHideController{}

	// shortcutService = nil → 跳过自动隐藏
	a := &App{
		windowVisibility: window.NewWindowVisibilityManagerWithController(
			func() platform.WindowHandle { return platform.WindowHandle(0x1234) },
			fc,
		),
		shortcutService: nil, // 无恢复路径
	}

	a.applyOutputModeWindowPolicy("desktop", "mobile")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("shortcut=nil 时不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-12: windowVisibility 为 nil 时不 panic ---

func TestP4NoVisibilityManager(t *testing.T) {
	a := &App{
		windowVisibility: nil,
		shortcutService: shortcut.NewService(
			&fakeDelegate{},
			map[string]shortcut.KeyBinding{},
			func(callback func(map[string]shortcut.KeyBinding)) {},
		),
	}

	// 不应 panic
	a.applyOutputModeWindowPolicy("desktop", "mobile")
}

// ========================================================================
// Phase 4.1：Mobile Server 不可用时的行为
// ========================================================================

// --- T-P4-13: Startup mobile + server unavailable → Hide count = 0 ---

func TestP4StartupMobileNoServerNoHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc) // 无 mobileBridge/mobileServer → mobileOutputEnabled() = false

	a.applyOutputModeWindowPolicy("desktop", "mobile")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("server 不可用时不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-14: Runtime desktop → mobile + server unavailable → Hide count = 0 ---

func TestP4RuntimeDesktopToMobileNoServerNoHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc)

	a.applyOutputModeWindowPolicy("desktop", "mobile")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("desktop→mobile + server 不可用不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-15: Runtime dual → mobile + server unavailable → Hide count = 0 ---

func TestP4RuntimeDualToMobileNoServerNoHide(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc)

	a.applyOutputModeWindowPolicy("dual", "mobile")

	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("dual→mobile + server 不可用不应 Hide，实际 %d", hides)
	}
}

// --- T-P4-16: Server unavailable skip 后，仍为 mobile → 后续普通 callback 不补 Hide ---
//
// 场景：
//   1. desktop → mobile + server unavailable → skip（0 Hide）
//   2. 仍为 mobile → 同一 callback（mobile→mobile）→ 不补 Hide
func TestP4MobileNoServerNoCompensate(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc)

	// 1. 第一次 transition → skip
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("server 不可用不应 Hide，实际 %d", hides)
	}

	// 2. 仍为 mobile → 不补 Hide（one-shot 语义 + server 不可用）
	a.applyOutputModeWindowPolicy("mobile", "mobile")
	_, hides2 := fc.calls()
	if hides2 != 0 {
		t.Fatalf("mobile→mobile 后不应 Hide，实际 %d", hides2)
	}
}

// --- T-P4-17: desktop→mobile skip(server unavailable) → desktop → desktop→mobile(server available) → Hide once ---
//
// 场景：
//   1. desktop → mobile + server unavailable → skip
//   2. mobile → desktop
//   3. desktop → mobile + server available → Hide exactly once
func TestP4MobileNoServerThenAvailableHides(t *testing.T) {
	fc := &fakeHideController{}
	a := newPhase4App(fc) // 初始无 server

	// 1. skip
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	_, hides := fc.calls()
	if hides != 0 {
		t.Fatalf("首次 desktop→mobile + server 不可用不应 Hide，实际 %d", hides)
	}

	// 2. 切回 desktop
	a.applyOutputModeWindowPolicy("mobile", "desktop")

	// 3. 装配 server
	bridge, srv := startTestMobileServer(t)
	a.mobileBridge = bridge
	a.mobileServer = srv

	// 4. desktop → mobile + server available → Hide once
	a.applyOutputModeWindowPolicy("desktop", "mobile")
	shows, hides2 := fc.calls()
	if hides2 != 1 {
		t.Fatalf("第二次 desktop→mobile + server 可用应 Hide 1 次，实际 %d", hides2)
	}
	if shows != 0 {
		t.Fatalf("不应自动 Show，实际 %d", shows)
	}
}
