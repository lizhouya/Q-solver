package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/resume"
	"Q-Solver/pkg/screen"
	"Q-Solver/pkg/shortcut"
	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/state"
	"Q-Solver/pkg/task"
	"Q-Solver/pkg/tray"
	"Q-Solver/pkg/window"
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	configManager *config.ConfigManager
	stateManager  *state.StateManager
	taskManager   *task.TaskCoordinator

	llmService      *llm.Service
	resumeService   *resume.Service
	shortcutService *shortcut.Service
	screenService   *screen.Service
	solver          *solution.Solver

	// Mobile Output（001 变更）：事件镜像 + 局域网 SSE 服务
	mobileBridge *mobile.Bridge
	mobileServer *mobile.Server

	// Window Visibility（003 Phase 1）：窗口实体可见性管理器
	windowVisibility *window.WindowVisibilityManager

	// System Tray（003 Phase 3）：系统托盘（左键恢复 + 右键退出）
	trayManager *tray.TrayManager

	// mobileOutputModeCache 缓存启动时的 OutputMode，用于检测运行时切换
	mobileOutputModeCache string
}

func NewApp() *App {
	configManager := config.NewConfigManager()

	app := &App{
		configManager: configManager,
		stateManager:  state.NewStateManager(),
		taskManager:   task.NewTaskCoordinator(),
		screenService: screen.NewService(),
	}

	// Window Visibility（003 Phase 1）：复用 state manager 的 HWND，不重复创建句柄
	app.windowVisibility = window.NewWindowVisibilityManager(app.stateManager.GetHwnd)

	return app
}

// setupTray 创建并启动系统托盘（003 Phase 3）。
// 失败仅降级（记录日志），绝不影响主流程；托盘只是恢复入口（D-003-003）。
func (a *App) setupTray() {
	a.trayManager = tray.NewTrayManager(a.trayHandlers())
	if err := a.trayManager.Start(); err != nil {
		logger.Printf("系统托盘启动失败（降级：可仅用快捷键恢复）: %v", err)
		a.trayManager = nil
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.configManager.Load(); err != nil {
		logger.Printf("加载配置失败: %v", err)
	}

	cfg := a.configManager.Get()
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		runtime.WindowSetSize(ctx, cfg.WindowWidth, cfg.WindowHeight)
		logger.Printf("应用保存的窗口尺寸: %dx%d", cfg.WindowWidth, cfg.WindowHeight)
	}

	a.stateManager.Startup(ctx, a.EmitEvent)
	a.screenService.Startup(ctx)

	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	a.solver = solution.NewSolver(a.llmService.GetProvider())
	a.resumeService = resume.NewService(a.configManager.Get(), a.configManager)

	a.shortcutService = shortcut.NewService(a, a.configManager.Get().Shortcuts, func(callback func(map[string]shortcut.KeyBinding)) {
		a.configManager.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
			callback(newConfig.Shortcuts)
		})
	})
	a.shortcutService.Start()

	a.configManager.Subscribe(a.onConfigChanged)
	a.stateManager.UpdateInitStatus(state.StatusReady)

	// Mobile Output（附加能力）：失败仅降级，绝不阻断主逻辑
	a.setupMobile()

	// System Tray（003 Phase 3）：托盘恢复入口（失败仅降级）
	a.setupTray()

	// Phase 4：OutputMode=mobile 时自动隐藏桌面窗口（需在恢复路径装配之后）
	a.mobileOutputModeCache = a.mobileOutputMode()
	if a.mobileOutputModeCache == "mobile" {
		a.applyOutputModeWindowPolicy("desktop", "mobile")
	}
}

func (a *App) onConfigChanged(newConfig config.Config, oldConfig config.Config) {
	if a.solver != nil {
		a.solver.SetProvider(a.llmService.GetProvider())
	}

	if !newConfig.KeepContext && a.solver != nil {
		a.solver.ClearHistory()
	}

	if a.mobileBridge != nil {
		a.mobileBridge.SetModel(newConfig.Model)
	}

	// Phase 4：OutputMode 从非 mobile → mobile 时自动隐藏桌面窗口
	a.applyOutputModeWindowPolicy(oldConfig.OutputMode, newConfig.OutputMode)

	logger.Println("配置已更新并应用")
}

// applyOutputModeWindowPolicy 根据 OutputMode 切换自动隐藏/恢复桌面窗口（Phase 4）。
//
// 语义（transition-triggered action）：
//   - 仅当 previousMode != mobile && newMode == mobile 时调用一次 Hide()
//   - mobile → desktop/dual 不自动 Show（OutputMode 与 WindowVisible 是独立状态）
//   - 用户在 mobile 模式手动 Show() 后，不会因为当前 mode 仍是 mobile 再次 Hide
//     （因 onlyMobileEnter 逻辑确保同一 entry 只触发一次）
//
// 恢复路径保证：
//   - Ctrl+Shift+H（shortcut）和 Tray「显示 Q-Solver」是已验证的恢复入口
//   - 自动隐藏必须发生在这些恢复路径已装配之后（启动顺序保证：shortcut → tray → auto-hide）
//
// Mobile Server 可用性（Phase 4.1）：
//   - 无论 startup 还是 runtime，只有 Mobile Server 可用时才自动 Hide
//   - Server 不可用时跳过，记录 warning（不 panic/exit）
//   - 服务稍后恢复不补偿 Hide；仅在新的 non-mobile→mobile transition 再次判断
//
// 禁止：
//   - 轮询（仅 transition-triggered，不循环检测）
//   - 在 pkg/config 中 import pkg/window（依赖方向 app → config 和 app → window）
func (a *App) applyOutputModeWindowPolicy(previousMode, newMode string) {
	if newMode != "mobile" {
		// mobile → desktop/dual 不自动 Show（Phase 4 局部设计决策）
		return
	}
	if previousMode == "mobile" {
		// 已在 mobile 模式，不重复 Hide（one-shot 语义）
		return
	}

	// Phase 4.1：只有 Mobile Output 服务已可用时才自动隐藏
	if !a.mobileOutputEnabled() {
		logger.Println("mobile output unavailable; skip automatic desktop hide")
		return
	}

	// 确保至少一条恢复路径已装配
	if a.shortcutService == nil {
		logger.Println("自动隐藏跳过：快捷键服务未启动（无恢复路径）")
		return
	}

	if a.windowVisibility == nil {
		logger.Println("自动隐藏跳过：窗口可见性管理器未初始化")
		return
	}

	if err := a.windowVisibility.Hide(); err != nil {
		logger.Printf("自动隐藏桌面窗口失败: %v", err)
		// 失败时保持窗口显示，不 panic
		return
	}
	logger.Println("OutputMode=mobile：桌面窗口已自动隐藏")
}

func (a *App) OnShutdown(ctx context.Context) {
	if a.shortcutService != nil {
		a.shortcutService.Stop()
	}
	// System Tray（003 Phase 3）：先停止托盘（结束消息循环、删图标）
	if a.trayManager != nil {
		if err := a.trayManager.Stop(); err != nil {
			logger.Printf("停止托盘失败: %v", err)
		}
	}
	a.shutdownMobile() // 关闭 SSE 服务，释放端口（001）
	if err := a.configManager.Save(); err != nil {
		logger.Printf("保存配置失败: %v", err)
	}
}

func (a *App) EmitEvent(eventName string, data ...interface{}) {
	// 约束一：解题输出事件 由 OutputMode Router 分流；其余事件永远走桌面
	if isSolutionOutputEvent(eventName) {
		a.routeSolutionEvent(eventName, data...)
		return
	}
	runtimeEmit(a.ctx, eventName, data...)
}

// runtimeEmit 封装 Wails runtime 事件发射（避免在每个调用点 import runtime）。
func runtimeEmit(ctx context.Context, eventName string, data ...interface{}) {
	if ctx == nil {
		return
	}
	runtime.EventsEmit(ctx, eventName, data...)
}

// isSolutionOutputEvent 判断事件是否属于「解题输出事件白名单」（见 app/mobile.go）。
func isSolutionOutputEvent(event string) bool {
	return solutionOutputEvents[event]
}

// Show 恢复并聚焦主窗口（003 Phase 3.1 一致性收口）。
//
// 语义（两步，职责分离）：
//  1. 状态一致性：统一经 WindowVisibilityManager.Show() —— 若当前 WindowHidden，
//     则由状态机恢复为 Visible（实体 SW_SHOW + 状态同步）；若已 Visible 则幂等返回，
//     不重复触发平台调用。保证所有「Hidden → Visible」业务入口都同步状态机。
//  2. 前台聚焦：调用 Wails runtime.WindowShow 把窗口带回前台（聚焦职责归属 Wails，
//     WindowVisibilityManager 不承担聚焦）。
//
// 调用方：SecondInstance 唤起（main.go）、托盘左键/菜单显示（trayOnShow）。
func (a *App) Show() {
	if a.windowVisibility != nil {
		if err := a.windowVisibility.Show(); err != nil {
			logger.Printf("恢复窗口显示失败: %v", err)
		}
	}
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
}
