package app

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/shortcut"
)

// TestDecideRoute 覆盖 OutputMode Router 全部分支（02-plan §2.1 约束一 + §2.2 约束二降级）。
func TestDecideRoute(t *testing.T) {
	cases := []struct {
		name            string
		mode            string
		serverAvailable bool
		wantDesktop     bool
		wantMobile      bool
	}{
		{"desktop → 仅桌面", "desktop", true, true, false},
		{"desktop + 不可用 → 仍桌面", "desktop", false, true, false},
		{"mobile + 可用 → 仅手机", "mobile", true, false, true},
		{"mobile + 不可用 → 降级桌面", "mobile", false, true, false},
		{"dual + 可用 → 双端", "dual", true, true, true},
		{"dual + 不可用 → 桌面仍渲染", "dual", false, true, true},
		{"未知 mode → 兜底桌面", "", true, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideRoute(c.mode, c.serverAvailable)
			if got.needDesktop != c.wantDesktop || got.needMobile != c.wantMobile {
				t.Errorf("%s: desktop=%v mobile=%v，期望 desktop=%v mobile=%v",
					c.name, got.needDesktop, got.needMobile, c.wantDesktop, c.wantMobile)
			}
		})
	}
}

func TestDecideScrollRoute(t *testing.T) {
	cases := []struct {
		name            string
		mode            string
		serverAvailable bool
		wantDesktop     bool
		wantMobile      bool
	}{
		{"desktop + 可用 → 仅桌面", "desktop", true, true, false},
		{"mobile + 可用 → 仅手机", "mobile", true, false, true},
		{"dual + 可用 → 双端", "dual", true, true, true},
		{"mobile + 不可用 → 回退桌面", "mobile", false, true, false},
		{"dual + 不可用 → 回退桌面", "dual", false, true, false},
		{"未知 mode → 仅桌面", "unknown", true, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideScrollRoute(c.mode, c.serverAvailable)
			if got.needDesktop != c.wantDesktop || got.needMobile != c.wantMobile {
				t.Errorf("desktop=%v mobile=%v，期望 desktop=%v mobile=%v",
					got.needDesktop, got.needMobile, c.wantDesktop, c.wantMobile)
			}
		})
	}
}

func TestMobileShortcutHintsOnlyExposeLabelsAndDisplayNames(t *testing.T) {
	bindings := map[string]shortcut.KeyBinding{
		"screenshot":  {ComboID: "119", KeyName: "F8"},
		"scroll_up":   {ComboID: "33+164", KeyName: "Alt+PgUp"},
		"scroll_down": {ComboID: "34+164", KeyName: "Alt+PgDn"},
		"send":        {ComboID: "SECRET-VK", KeyName: "Ctrl+J"},
	}

	hints := mobileShortcutHints(bindings)
	if len(hints) != 4 {
		t.Fatalf("应只返回已配置且有显示名的 4 条提示，实际 %+v", hints)
	}
	if hints[0] != (mobile.ShortcutHint{Label: "截图", KeyName: "F8"}) {
		t.Fatalf("首条提示不正确: %+v", hints[0])
	}
	if hints[2] != (mobile.ShortcutHint{Label: "向上滚动", KeyName: "Alt+PgUp"}) ||
		hints[3] != (mobile.ShortcutHint{Label: "向下滚动", KeyName: "Alt+PgDn"}) {
		t.Fatalf("滚动提示不正确: %+v", hints)
	}

	raw, err := json.Marshal(hints)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(raw)
	for _, forbidden := range []string{"vkCode", "comboID", "33+164", "34+164", "SECRET-VK", "token", "path"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("Mobile hints 不得包含 %q，实际 %s", forbidden, serialized)
		}
	}
}

// TestSolutionOutputEvents 确认白名单事件存在（与 pkg/solution 实际事件名对齐）。
func TestSolutionOutputEvents(t *testing.T) {
	expect := []string{
		"solution-stream-start",
		"solution-stream-thinking",
		"solution-stream-chunk",
		"solution",
		"solution-error",
	}
	for _, e := range expect {
		if !isSolutionOutputEvent(e) {
			t.Errorf("%s 应在白名单内", e)
		}
	}
	// 非解题事件绝不能被误判为输出事件
	nonOutput := []string{"toast", "start-solving", "user-message", "screenshot-taken", "init-status"}
	for _, e := range nonOutput {
		if isSolutionOutputEvent(e) {
			t.Errorf("%s 不应在白名单内", e)
		}
	}
}

// TestGetMobileOutputInfoUnavailable 验证 mobile server 不存在 / 未就绪时的返回：
// Available=false, Port=0（行为保持不变，004 不回归）。
func TestGetMobileOutputInfoUnavailable(t *testing.T) {
	a := &App{} // mobileBridge / mobileServer 均为 nil → mobileOutputEnabled() = false

	info := a.GetMobileOutputInfo()
	if info.Available {
		t.Fatalf("server 不可用时 Available 应为 false，实际 true")
	}
	if info.Port != 0 {
		t.Fatalf("server 不可用时 Port 应为 0，实际 %d", info.Port)
	}
	if info.URL != "" {
		t.Fatal("server 不可用时 URL 应为空")
	}
	if info.Status != mobileStatusNotStarted || !info.Retryable {
		t.Fatalf("未启动状态应可重试，实际 status=%q retryable=%v", info.Status, info.Retryable)
	}
}

// TestGetMobileOutputInfoAvailable 验证 mobile server 就绪时：
// Available=true、URL 存在、Port 返回真实端口（默认 38511，来自 Server.Port()，非硬编码）。
//
// mobileServer 为具体类型 *mobile.Server、无 mock 接口，因此沿用 phase4_test 的既有做法：
// 启动真实 Server（默认端口），断言 DefaultPort。若端口 38511 冲突则自动 skip。
func TestGetMobileOutputInfoAvailable(t *testing.T) {
	bridge, err := mobile.NewBridge(0)
	if err != nil {
		t.Fatal("创建 mobile bridge 失败")
	}
	srv, err := mobile.NewServer(bridge, appTestMobileToken)
	if err != nil {
		t.Fatalf("创建 mobile server 失败: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Skipf("启动 mobile server 失败（默认端口 38511 可能冲突）: %v", err)
	}
	t.Cleanup(func() { srv.Stop() })

	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		Generation: 1,
	}}
	a := &App{
		mobileBridge:       bridge,
		mobileServer:       srv,
		mobileStateManager: manager,
		mobileRuntimeState: manager.state,
		mobileStateReady:   true,
		mobileStatus:       mobileStatusReady,
	}

	got := a.GetMobileOutputInfo()
	if !got.Available {
		t.Fatalf("server 就绪时 Available 应为 true，实际 false")
	}
	if got.URL == "" {
		t.Fatalf("server 就绪时 URL 不应为空")
	}
	if got.Port != mobile.DefaultPort {
		t.Fatalf("Port 应为 %d，实际 %d", mobile.DefaultPort, got.Port)
	}
	if got.Status != mobileStatusReady || got.Retryable {
		t.Fatalf("就绪状态不应可重试，实际 status=%q retryable=%v", got.Status, got.Retryable)
	}

	beforeURL := srv.URL()
	retried := a.RetryMobileOutput()
	if !retried.Available || srv.URL() != beforeURL {
		t.Fatal("运行中的 Mobile 服务重试应幂等且保留连接地址")
	}
}

func TestRetryMobileOutputAfterPortBecomesAvailable(t *testing.T) {
	listener, err := net.Listen("tcp", "0.0.0.0:38511")
	if err != nil {
		t.Skipf("无法占用默认端口以建立测试前提: %v", err)
	}

	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		Generation: 1,
	}}
	a := &App{mobileStateManager: manager}
	a.setupMobile()
	info := a.GetMobileOutputInfo()
	if info.Available || info.Status != mobileStatusPortUnavailable || !info.Retryable {
		listener.Close()
		t.Fatalf("端口占用时状态不正确: available=%v status=%q retryable=%v", info.Available, info.Status, info.Retryable)
	}
	if info.URL != "" || info.Port != 0 {
		listener.Close()
		t.Fatal("不可用状态不得返回连接地址或端口")
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("释放测试端口失败: %v", err)
	}
	retried := a.RetryMobileOutput()
	t.Cleanup(func() { a.shutdownMobile() })
	if !retried.Available || retried.Status != mobileStatusReady {
		t.Fatalf("释放端口后重试应恢复服务: available=%v status=%q", retried.Available, retried.Status)
	}
}

func TestRetryMobileOutputDoesNotHideCurrentWindow(t *testing.T) {
	controller := &fakeHideController{}
	a := newPhase4AppWithMobile(t, controller)

	info := a.RetryMobileOutput()
	if !info.Available {
		t.Fatal("测试前提：Mobile 服务应保持可用")
	}
	_, hides := controller.calls()
	if hides != 0 {
		t.Fatalf("重试 Mobile 服务不得触发桌面隐藏，实际 Hide %d 次", hides)
	}
}

func TestMobileHintsIncludeActualClearAndNextBindings(t *testing.T) {
	bindings := map[string]shortcut.KeyBinding{
		"clear_mobile_display": {KeyName: "Ctrl+Alt+C"},
		"start_next_exam":      {KeyName: "Ctrl+Alt+N"},
	}
	hints := mobileShortcutHints(bindings)
	if len(hints) != 2 || hints[0].KeyName != "Ctrl+Alt+C" || hints[1].KeyName != "Ctrl+Alt+N" {
		t.Fatalf("missing actual bindings: %+v", hints)
	}
	bindings["start_next_exam"] = shortcut.KeyBinding{}
	if len(mobileShortcutHints(bindings)) != 1 {
		t.Fatal("empty shortcut shown")
	}
}
