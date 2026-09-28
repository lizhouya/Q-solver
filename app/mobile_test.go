package app

import (
	"testing"
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