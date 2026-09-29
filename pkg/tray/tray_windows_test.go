//go:build windows

package tray

import (
	"testing"
)

// TestDispatchMenu 覆盖托盘右键菜单「显示/隐藏/退出」三向分派：
//   - MenuShow → OnShow 回调恰好一次
//   - MenuHide → OnHide 回调恰好一次
//   - MenuExit → OnExit 回调恰好一次
//
// 该测试不触碰真实 Win32（ShowTrayMenu 在 onRightClick 中调用，此处直接测 ID 分派层）。
func TestDispatchMenu(t *testing.T) {
	var showCalls, hideCalls, exitCalls int

	c := &windowsTrayController{
		handlers: TrayEventHandlers{
			OnShow: func() { showCalls++ },
			OnHide: func() { hideCalls++ },
			OnExit: func() { exitCalls++ },
		},
	}

	// 三选一项都应触发对应回调且互不干扰
	c.dispatchMenu(MenuShow)
	c.dispatchMenu(MenuHide)
	c.dispatchMenu(MenuExit)
	if showCalls != 1 || hideCalls != 1 || exitCalls != 1 {
		t.Fatalf("MenuShow/Hide/Exit 应各触发 1 次，实际 show=%d hide=%d exit=%d",
			showCalls, hideCalls, exitCalls)
	}

	// 无法识别的 ID → 静默忽略（不会误触发任何回调）
	showCalls, hideCalls, exitCalls = 0, 0, 0
	c.dispatchMenu(0xFFFF)
	if showCalls != 0 || hideCalls != 0 || exitCalls != 0 {
		t.Fatalf("未知菜单 ID 不应触发任何回调，实际 show=%d hide=%d exit=%d",
			showCalls, hideCalls, exitCalls)
	}
}
