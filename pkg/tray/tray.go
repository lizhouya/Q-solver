// Package tray 提供系统托盘管理（Change 003 Phase 3）。
//
// 职责：与 pkg/window 对称——TrayManager 只负责平台无关的生命周期
// （Start/Stop 幂等、回调分派），实际的 Win32 托盘操作交给底层 TrayController
// 抽象（Windows 真机见 tray_windows.go，测试用 mock 见 tray_test.go）。
//
// 设计要点：
//   - 托盘与窗口实体可见性（pkg/window）、GhostMode（pkg/state）完全独立；
//   - 托盘生命周期不阻塞 Wails Startup：Start 不卡主线程，Stop 能结束消息循环；
//   - 失败仅降级（记录日志），绝不 panic / 退出（见 00-request §3 容错要求）。
package tray

import "errors"

// 托盘右键菜单项 ID（与 tray_windows.go 构建的菜单一一对应）。
const (
	// MenuShow 显示 Q-Solver
	MenuShow uint32 = 1
	// MenuHide 隐藏 Q-Solver
	MenuHide uint32 = 2
	// MenuExit 退出应用
	MenuExit uint32 = 3
)

// TrayEventHandlers 托盘事件回调集，由 app 层注入具体动作。
type TrayEventHandlers struct {
	// OnShow 左键单击 / 菜单「显示 Q-Solver」触发
	OnShow func()
	// OnHide 菜单「隐藏 Q-Solver」触发
	OnHide func()
	// OnExit 菜单「退出」触发
	OnExit func()
}

// TrayController 抽象「系统托盘」的平台操作。
// 真机（Windows）由 tray_windows.go 实现；单元测试用 mock（见 tray_test.go）。
type TrayController interface {
	// Start 创建托盘图标并启动消息循环（内部 goroutine，不阻塞调用方）。
	// handlers 提供托盘事件回调；回调可能从消息循环所在 goroutine 触发，
	// 实现需保证回调线程安全。
	Start(handlers TrayEventHandlers) error
	// Stop 删除托盘图标并结束消息循环，幂等。
	Stop() error
}

// currentController 由平台文件在 init() 中绑定（windows → windowsController；darwin → darwinController）。
var currentController = func() TrayController {
	return &notImplementedController{}
}

// notImplementedController 兜底实现：无平台文件绑定时的占位（理论上不会触发）。
type notImplementedController struct{}

func (*notImplementedController) Start(_ TrayEventHandlers) error {
	return errors.New("托盘控制器未初始化（平台未绑定）")
}

func (*notImplementedController) Stop() error {
	return errors.New("托盘控制器未初始化（平台未绑定）")
}

// TrayManager 系统托盘管理器。职责单一：持有 TrayController 并保证 Start/Stop 幂等。
type TrayManager struct {
	controller TrayController
	handlers   TrayEventHandlers
	started    bool
}

// NewTrayManager 创建托盘管理器。handlers 在 Start 时传递给底层控制器。
func NewTrayManager(handlers TrayEventHandlers) *TrayManager {
	return &TrayManager{
		controller: currentController(),
		handlers:   handlers,
	}
}

// Start 启动托盘。已启动时为幂等操作（不重复调用底层）。
func (m *TrayManager) Start() error {
	if m.started {
		return nil
	}
	if err := m.controller.Start(m.handlers); err != nil {
		return err
	}
	m.started = true
	return nil
}

// Stop 停止托盘并释放资源。未启动时为幂等操作。
func (m *TrayManager) Stop() error {
	if !m.started {
		return nil
	}
	err := m.controller.Stop()
	// 即使底层 Stop 失败也标记未启动，允许重试；错误如实上报。
	m.started = false
	return err
}

// IsRunning 报告托盘是否已启动。
func (m *TrayManager) IsRunning() bool {
	return m.started
}
