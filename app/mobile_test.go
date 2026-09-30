package app

import (
	"testing"

	"Q-Solver/pkg/mobile"
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
		t.Fatalf("server 不可用时 URL 应为空，实际 %q", info.URL)
	}
}

// TestGetMobileOutputInfoAvailable 验证 mobile server 就绪时：
// Available=true、URL 存在、Port 返回真实端口（默认 38511，来自 Server.Port()，非硬编码）。
//
// mobileServer 为具体类型 *mobile.Server、无 mock 接口，因此沿用 phase4_test 的既有做法：
// 启动真实 Server（默认端口），断言 DefaultPort。若端口 38511 冲突则自动 skip。
func TestGetMobileOutputInfoAvailable(t *testing.T) {
	bridge := mobile.NewBridge()
	srv, err := mobile.NewServer(bridge)
	if err != nil {
		t.Fatalf("创建 mobile server 失败: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Skipf("启动 mobile server 失败（默认端口 38511 可能冲突）: %v", err)
	}
	t.Cleanup(func() { srv.Stop() })

	a := &App{mobileBridge: bridge, mobileServer: srv}

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
}