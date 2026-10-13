package app

import "Q-Solver/pkg/platform"

// ToggleVisibility 切换可见性
func (a *App) ToggleVisibility() {
	a.stateManager.ToggleVisibility()
}

// ToggleClickThrough 切换鼠标穿透，返回平台调用成功后的真实状态。
func (a *App) ToggleClickThrough() (bool, error) {
	return a.stateManager.ToggleClickThrough()
}

// MoveWindow 移动窗口
func (a *App) MoveWindow(dx, dy int) {
	a.stateManager.MoveWindow(dx, dy)
}

// RestoreFocus 恢复焦点
func (a *App) RestoreFocus() {
	a.stateManager.RestoreFocus()
}

// RemoveFocus 移除焦点
func (a *App) RemoveFocus() {
	a.stateManager.RemoveFocus()
}

// SetWindowAlwaysOnTop 设置窗口是否置顶
func (a *App) SetWindowAlwaysOnTop(alwaysOnTop bool) {
	hwnd := a.stateManager.GetHwnd()
	if hwnd == 0 {
		return
	}
	if alwaysOnTop {
		platform.SetWindowLevel(hwnd, platform.WindowLevelFloating)
	} else {
		platform.SetWindowLevel(hwnd, platform.WindowLevelNormal)
	}
}

// ==================== Window Visibility（003 Phase 1） ====================

// HideWindow 隐藏主窗口（实体隐藏，非隐身/防录屏）。
func (a *App) HideWindow() error {
	return a.windowVisibility.Hide()
}

// ShowWindow 显示主窗口（实体显示）。
func (a *App) ShowWindow() error {
	return a.windowVisibility.Show()
}

// ToggleWindowVisible 切换主窗口显示/隐藏，返回切换后的状态字符串。
func (a *App) ToggleWindowVisible() (string, error) {
	state, err := a.windowVisibility.Toggle()
	if err != nil {
		return "", err
	}
	return state.String(), nil
}

// WindowVisible 返回主窗口当前是否可见。
func (a *App) WindowVisible() bool {
	return a.windowVisibility.IsVisible()
}
