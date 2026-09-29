//go:build darwin

package window

import (
	"errors"

	"Q-Solver/pkg/platform"
)

// darwinController 当前为一空实现（003 首期仅支持 Windows，D-003-007）。
type darwinController struct{}

// init 将 macOS 占位控制器绑定到 currentController 工厂。
func init() {
	currentController = func() WindowController {
		return &darwinController{}
	}
}

// Show 实体显示窗口（macOS 未实现）
func (c *darwinController) Show(hwnd platform.WindowHandle) error {
	return errors.New("实体显示/隐藏仅支持 Windows（003 非目标）")
}

// Hide 实体隐藏窗口（macOS 未实现）
func (c *darwinController) Hide(hwnd platform.WindowHandle) error {
	return errors.New("实体显示/隐藏仅支持 Windows（003 非目标）")
}