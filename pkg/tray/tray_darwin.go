//go:build darwin

package tray

import "errors"

// 003 变更仅支持 Windows（D-003-007）：macOS 托盘不做实现，占位返回错误保证跨平台编译。

type darwinTrayController struct{}

func init() {
	currentController = func() TrayController {
		return &darwinTrayController{}
	}
}

func (*darwinTrayController) Start(_ TrayEventHandlers) error {
	return errors.New("系统托盘未实现（003 仅支持 Windows）")
}

func (*darwinTrayController) Stop() error {
	return nil
}

var _ TrayController = (*darwinTrayController)(nil)
