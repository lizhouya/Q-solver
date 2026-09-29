package app

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/tray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ==================== System Tray（003 Phase 3） ====================

// trayHandlers 返回注入给 TrayManager 的事件回调集。
func (a *App) trayHandlers() tray.TrayEventHandlers {
	return tray.TrayEventHandlers{
		OnShow: a.trayOnShow,
		OnHide: a.trayOnHide,
		OnExit: a.trayOnExit,
	}
}

// trayOnShow 托盘左键/菜单「显示 Q-Solver」→ 复用 app.Show()：
//  1. WindowVisibilityManager.Show()（Hidden → Visible，状态机同步；已可见则幂等）；
//  2. Wails runtime.WindowShow（前台聚焦）。
//
// 托盘回调来自消息循环 goroutine，本方法必须线程安全（WindowVisibilityManager 内已加锁）。
func (a *App) trayOnShow() {
	a.Show()
}

// trayOnHide 托盘菜单「隐藏 Q-Solver」→ WindowVisibilityManager.Hide()（状态机同步）。
// 禁止直接 ShowWindow(SW_HIDE)——实体隐藏的唯一业务入口是 WindowVisibilityManager。
func (a *App) trayOnHide() {
	if a.windowVisibility == nil {
		return
	}
	if err := a.windowVisibility.Hide(); err != nil {
		logger.Printf("托盘隐藏窗口失败: %v", err)
	}
}

// trayOnExit 托盘「退出」：优雅退出，触发 OnShutdown 既有清理链。
func (a *App) trayOnExit() {
	logger.Println("托盘菜单退出")
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
