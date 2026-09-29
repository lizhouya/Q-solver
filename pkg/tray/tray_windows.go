//go:build windows

package tray

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// 本文件是 TrayController 的 Windows 真机实现：
//   - 独立 goroutine + runtime.LockOSThread 确保「创建窗口的线程」即「消息循环线程」
//     （Win32 要求：窗口消息只能由拥有线程分派，否则回调收不到）；
//   - 消息循环不阻塞 Wails Startup；Stop 通过向窗口投递停止消息让 GetMessage 返回 0 退出；
//   - wndProc 分派：左键 → OnShow；右键 → 弹出菜单并按 ID 回调。

// 托盘宿主窗口类名（进程内唯一）。
const trayClassName = "Q-SolverTrayHostClass"

const (
	// trayIconID 托盘图标唯一 ID（同宿主窗口内）。
	trayIconID uint32 = 1
	// trayCallbackMessage 托盘回调消息（WM_USER 保留段的私有消息）。
	trayCallbackMessage = 0x0400 + 1
	// trayStopMessage 停止消息（让消息循环退出）。
	trayStopMessage = 0x0400 + 2
	// trayTooltip 托盘悬停提示。
	trayTooltip = "Q-Solver"
)

// WM_CONTEXTMENU（Windows）：右键菜单独有通知，用于支持辅助功能。
const wmContextMenu = 0x007B

// activeController 当前托盘控制器（单实例；wndProc 经它分派回调）。
// 用 sync 保护，Stop 后可置空，避免悬挂回调调用已释放实例。
var (
	activeMu         sync.RWMutex
	activeController *windowsTrayController
)

// trayWndProc 宿主窗口过程：分发托盘回调消息与停止消息。
// syscall.NewCallback 支持 4 个 uintptr 参数 + uintptr 返回值（WNDPROC 签名）。
var trayWndProc = syscall.NewCallback(func(h uintptr, msg uintptr, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case trayCallbackMessage:
		activeMu.RLock()
		c := activeController
		activeMu.RUnlock()
		if c == nil {
			return 0
		}
		switch uint32(lParam) {
		case platform.WM_LBUTTONUP, wmContextMenu:
			// 左键单击 → 显示主窗口
			c.onShow()
		case platform.WM_RBUTTONUP:
			// 右键 → 弹出菜单
			c.onRightClick(h)
		}
		return 0
	case trayStopMessage:
		// 收到停止通知：让消息循环的 GetMessage 返回 0 退出
		platform.PostQuitMessage(0)
		return 0
	default:
		return platform.DefWindowProc(h, uint32(msg), wParam, lParam)
	}
})

// windowsTrayController 托盘控制器（Windows）。
type windowsTrayController struct {
	mu       sync.Mutex
	started  bool
	handlers TrayEventHandlers
	hwnd     uintptr

	// ready 就绪信号：run() 创建好图标后写入 nil，失败写 err（有缓冲，Start 等它）。
	ready chan error
	// stopped 停止信号：消息循环退出后 close。
	stopped chan struct{}
}

func init() {
	currentController = func() TrayController {
		return newWindowsController()
	}
}

// newWindowsController 构造 Windows 托盘控制器。
func newWindowsController() *windowsTrayController {
	return &windowsTrayController{
		ready:   make(chan error, 1),
		stopped: make(chan struct{}),
	}
}

func (c *windowsTrayController) Start(handlers TrayEventHandlers) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.handlers = handlers
	c.started = true
	c.mu.Unlock()

	// 独立 goroutine：拥有窗口 + 消息循环。
	go c.run()

	// 阻塞到就绪：run() 要么成功创建图标发 nil，要么失败发 err。
	// 该等待为毫秒级（注册类+建窗+NotifyIcon），不构成 Startup 阻塞。
	return <-c.ready
}

func (c *windowsTrayController) Stop() error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	hwnd := c.hwnd
	c.started = false
	c.mu.Unlock()

	if hwnd != 0 {
		// 通知消息循环退出（落到窗口线程，wndProc 里 PostQuitMessage）
		platform.PostMessage(hwnd, trayStopMessage, 0, 0)
	}
	select {
	case <-c.stopped:
	case <-time.After(2 * time.Second):
		logger.Println("托盘消息循环退出超时")
	}
	return nil
}

// run 在锁定的 OS 线程上创建宿主窗口并进入消息循环。goroutine 期间保持执行。
func (c *windowsTrayController) run() {
	// 关键：窗口创建与消息循环必须在同一 OS 线程
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.stopped)

	// run 的任何失败都必须复位 started 并推进 ready，保证可重试。
	fail := func(err error) {
		logger.Printf("托盘启动失败: %v", err)
		c.mu.Lock()
		c.started = false
		c.mu.Unlock()
		c.ready <- err
	}

	// 1. 注册窗口类（幂等）+ 创建宿主窗口
	if err := platform.RegisterTrayWindowClass(trayClassName, trayWndProc); err != nil {
		fail(err)
		return
	}
	hwnd, err := platform.CreateTrayHostWindow(trayClassName)
	if err != nil {
		fail(err)
		return
	}
	c.mu.Lock()
	c.hwnd = hwnd
	c.mu.Unlock()

	// 2. 关联控制器实例（供 wndProc 通过全局读取）
	activeMu.Lock()
	activeController = c
	activeMu.Unlock()

	// 3. 添加托盘图标（复用应用图标）
	icon := platform.LoadAppIcon()
	if err := platform.AddTrayIcon(hwnd, icon, trayTooltip, trayIconID, trayCallbackMessage); err != nil {
		activeMu.Lock()
		activeController = nil
		activeMu.Unlock()
		platform.DestroyTrayHostWindow(hwnd)
		fail(err)
		return
	}
	logger.Println("系统托盘已启动")
	c.ready <- nil // 就绪：Start 解除阻塞；此后 Stop 可安全投递停止消息

	// 4. 消息循环（阻塞直到收到停止消息）
	var msg platform.MSG
	for {
		ret := platform.GetMessage(&msg, hwnd, 0, 0)
		if ret <= 0 { // 0=退出/error；负值=错误码
			break
		}
		platform.TranslateMessage(&msg)
		platform.DispatchMessage(&msg)
	}

	// 5. 清理
	activeMu.Lock()
	activeController = nil
	activeMu.Unlock()
	platform.DeleteTrayIcon(hwnd, trayIconID)
	platform.DestroyTrayHostWindow(hwnd)
	c.mu.Lock()
	c.hwnd = 0
	c.mu.Unlock()
	logger.Println("托盘消息循环已退出")
}

// onShow 左键 → 显示主窗口回调。
func (c *windowsTrayController) onShow() {
	if cb := c.handlers.OnShow; cb != nil {
		cb()
	}
}

// onHide 菜单「隐藏 Q-Solver」→ 隐藏主窗口回调。
func (c *windowsTrayController) onHide() {
	if cb := c.handlers.OnHide; cb != nil {
		cb()
	}
}

// onRightClick 右键 → 弹出菜单，根据选中项分派。
func (c *windowsTrayController) onRightClick(hwnd uintptr) {
	menu := []platform.TrayMenuItem{
		{ID: MenuShow, Text: "显示 Q-Solver(&S)"},
		{ID: MenuHide, Text: "隐藏 Q-Solver(&H)"},
		{Separator: true},
		{ID: MenuExit, Text: "退出(&X)"},
	}
	id, err := platform.ShowTrayMenu(hwnd, menu)
	if err != nil {
		logger.Printf("托盘菜单出错: %v", err)
		return
	}
	c.dispatchMenu(id)
}

// dispatchMenu 根据菜单选中项 ID 分派动作（独立方法，便于单元测试）。
func (c *windowsTrayController) dispatchMenu(id uint32) {
	switch id {
	case MenuShow:
		c.onShow()
	case MenuHide:
		c.onHide()
	case MenuExit:
		if cb := c.handlers.OnExit; cb != nil {
			cb()
		}
	}
}

// 断言 panic-free 的编译期检查（无需实例）。
var _ TrayController = (*windowsTrayController)(nil)
