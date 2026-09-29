//go:build windows

package window

import "Q-Solver/pkg/platform"

// windowsController 封装 Windows ShowWindow，向 WindowVisibilityManager 提供真实实体窗口控制。
// 业务层（app / WindowVisibilityManager）不允许直接触碰 Win32，统一经 platform 封装。
type windowsController struct{}

// init 将当前平台的控制器绑定到 currentController（window.go）工厂。
func init() {
	currentController = func() WindowController {
		return &windowsController{}
	}
}

// Show 实体显示窗口
func (c *windowsController) Show(hwnd platform.WindowHandle) error {
	return platform.ShowWindow(hwnd, platform.SW_SHOW)
}

// Hide 实体隐藏窗口
func (c *windowsController) Hide(hwnd platform.WindowHandle) error {
	return platform.ShowWindow(hwnd, platform.SW_HIDE)
}