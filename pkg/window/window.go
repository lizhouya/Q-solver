// Package window 提供窗口「实体可见性」管理（Change 003，仅实现 Phase 1）。
//
// 职责：WindowVisibilityManager 是平台无关的窗口实体显示/隐藏管理器，
// 只负责状态机与调用点，实际的 Win32 操作交给底层 WindowController 抽象
// （Windows 真机见 window_windows.go，测试用 mock 见 window_test.go）。
//
// 三态独立性（003 设计约束）：
//   - GhostMode（防录屏隐身，F9）由 pkg/state 管理，本包不触碰；
//   - WindowVisible（实体可见性）由本包独立存储；
//   - OutputMode（desktop/mobile/dual）由 config 管理，本包不读不写。
package window

import (
	"errors"
	"sync"

	"Q-Solver/pkg/platform"
)

// WindowVisibleState 窗口「实体可见」状态（003：WindowVisibleState，独立于 GhostMode/OutputMode）。
type WindowVisibleState int

const (
	// WindowVisible 窗口可见（初始状态）
	WindowVisible WindowVisibleState = iota
	// WindowHidden 窗口隐藏
	WindowHidden
)

// String 返回可读状态名（供事件透传）。
func (s WindowVisibleState) String() string {
	if s == WindowVisible {
		return "visible"
	}
	return "hidden"
}

// WindowController 抽象「实体显示/隐藏」的平台操作。
// 真机（Windows）由 window_windows.go 实现；单元测试用 mock（见 window_test.go）。
type WindowController interface {
	// Show 实体显示窗口
	Show(hwnd platform.WindowHandle) error
	// Hide 实体隐藏窗口
	Hide(hwnd platform.WindowHandle) error
}

// currentController 由平台文件在 init() 中绑定（windows → windowsController；darwin → darwinController）。
var currentController = func() WindowController {
	return &notImplementedController{}
}

// notImplementedController 兜底实现：无平台文件绑定时的占位（理论上不会触发）。
type notImplementedController struct{}

func (c *notImplementedController) Show(hwnd platform.WindowHandle) error {
	return errors.New("窗口控制器未初始化（平台未绑定）")
}

func (c *notImplementedController) Hide(hwnd platform.WindowHandle) error {
	return errors.New("窗口控制器未初始化（平台未绑定）")
}

// WindowVisibilityManager 窗口可见性管理器（Change 003 Phase 1）。
type WindowVisibilityManager struct {
	mu sync.RWMutex

	state WindowVisibleState

	// controller 平台窗口操作（真机 / mock）
	controller WindowController
	// hwndProvider 复用上层窗口句柄（app 注入 state.StateManager.GetHwnd），避免重复存储 HWND
	hwndProvider func() platform.WindowHandle

	// visibilityChanged 状态变更回调（预留，Phase 4 前端事件用；当前可为 nil）
	visibilityChanged func(WindowVisibleState)
}

// NewWindowVisibilityManager 创建窗口可见性管理器。
// hwndProvider 由调用方注入窗口句柄来源（复用已有 state manager 的 HWND，不重复创建）。
func NewWindowVisibilityManager(hwndProvider func() platform.WindowHandle) *WindowVisibilityManager {
	return &WindowVisibilityManager{
		state:        WindowVisible,
		controller:   currentController(),
		hwndProvider: hwndProvider,
	}
}

// NewWindowVisibilityManagerWithController 创建窗口可见性管理器并显式注入控制器。
// 常规生产路径使用 NewWindowVisibilityManager（自动绑定当前平台控制器）；
// 本构造器仅为单元测试注入测试控制器而设（见 app 层托盘/二实例链路测试）。
func NewWindowVisibilityManagerWithController(hwndProvider func() platform.WindowHandle, c WindowController) *WindowVisibilityManager {
	return &WindowVisibilityManager{
		state:        WindowVisible,
		controller:   c,
		hwndProvider: hwndProvider,
	}
}

// SetVisibilityChange 注册可见性变更回调（Phase 4 在前端事件前预埋）。
func (m *WindowVisibilityManager) SetVisibilityChange(fn func(WindowVisibleState)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.visibilityChanged = fn
}

// IsVisible 报告当前窗口是否显示。
func (m *WindowVisibilityManager) IsVisible() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == WindowVisible
}

// State 返回当前实体可见状态（visible/hidden）。
func (m *WindowVisibilityManager) State() WindowVisibleState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// Hide 隐藏窗口。已隐藏时为幂等操作；窗口句柄不可用或平台调用失败时返回错误且不改状态。
func (m *WindowVisibilityManager) Hide() error {
	newState, err := m.transition(WindowHidden)
	if err != nil {
		return err
	}
	m.notify(newState)
	return nil
}

// Show 显示窗口。已显示时为幂等操作；句柄不可用或平台调用失败时返回错误。
func (m *WindowVisibilityManager) Show() error {
	newState, err := m.transition(WindowVisible)
	if err != nil {
		return err
	}
	m.notify(newState)
	return nil
}

// transition 执行状态迁移。
//   - 目标状态与当前一致 → 幂等返回（不触发平台调用、不触发回调）
//   - hwnd 不可用 / 控制器失败 → 返回错误，状态不变
//   - 成功 → 更新状态并返回新状态
func (m *WindowVisibilityManager) transition(target WindowVisibleState) (WindowVisibleState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == target {
		return m.state, nil
	}
	hwnd := m.hwnd()
	if hwnd == 0 {
		return m.state, errors.New("窗口句柄不可用")
	}
	var err error
	if target == WindowVisible {
		err = m.controller.Show(hwnd)
	} else {
		err = m.controller.Hide(hwnd)
	}
	if err != nil {
		return m.state, err
	}
	m.state = target
	return m.state, nil
}

// Toggle 切换可见状态（隐藏 ↔ 显示），返回新状态。
func (m *WindowVisibilityManager) Toggle() (WindowVisibleState, error) {
	if m.IsVisible() {
		err := m.Hide()
		return m.State(), err
	}
	err := m.Show()
	return m.State(), err
}

// hwnd 取窗口句柄（用户提供的 provider；未提供时视为不可用）。
func (m *WindowVisibilityManager) hwnd() platform.WindowHandle {
	if m.hwndProvider == nil {
		return 0
	}
	return m.hwndProvider()
}

// notify 在锁外触发可见性变更回调（保证顺序、避免持锁调用回调）。
func (m *WindowVisibilityManager) notify(state WindowVisibleState) {
	m.mu.RLock()
	fn := m.visibilityChanged
	m.mu.RUnlock()

	if fn == nil {
		return
	}
	fn(state)
}
