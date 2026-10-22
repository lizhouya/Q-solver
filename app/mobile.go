package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/shortcut"
	"errors"
)

// 解题输出事件白名单：仅这些事件受 OutputMode 路由控制（02-plan §2.1 约束一）。
// 其余所有事件（toast/screenshot/start-solving/user-message/shortcut/…）不经过路由，永远走桌面 Wails。
var solutionOutputEvents = map[string]bool{
	"solution-stream-start":    true, // 开始
	"solution-stream-thinking": true, // 思考链/分析过程
	"solution-stream-chunk":    true, // 正文流式
	"solution":                 true, // 最终答案
	"solution-error":           true, // 出错
}

type mobileShortcutHintDefinition struct {
	label   string
	actions []string
}

// mobileShortcutHintDefinitions 固定手机端可见的动作标签和顺序。
// actions 只用于本地查找配置，绝不写入 Snapshot；solve 是 macOS 对 screenshot 的兼容键。
var mobileShortcutHintDefinitions = []mobileShortcutHintDefinition{
	{label: "截图", actions: []string{"screenshot", "solve"}},
	{label: "发送解题", actions: []string{"send"}},
	{label: "删除截图", actions: []string{"delete"}},
	{label: "隐藏/显示", actions: []string{"toggle"}},
	{label: "隐藏窗口", actions: []string{"toggle_window"}},
	{label: "鼠标穿透", actions: []string{"clickthrough"}},
	{label: "向上移动", actions: []string{"move_up"}},
	{label: "向下移动", actions: []string{"move_down"}},
	{label: "向左移动", actions: []string{"move_left"}},
	{label: "向右移动", actions: []string{"move_right"}},
	{label: "向上滚动", actions: []string{"scroll_up"}},
	{label: "向下滚动", actions: []string{"scroll_down"}},
}

// mobileShortcutHints 从快捷键配置提取手机端展示数据。
// 返回值不包含 ComboID/VK、配置路径、Token 或其他配置字段。
func mobileShortcutHints(shortcuts map[string]shortcut.KeyBinding) []mobile.ShortcutHint {
	hints := make([]mobile.ShortcutHint, 0, len(mobileShortcutHintDefinitions))
	for _, definition := range mobileShortcutHintDefinitions {
		for _, action := range definition.actions {
			binding, ok := shortcuts[action]
			if !ok || binding.KeyName == "" {
				continue
			}
			hints = append(hints, mobile.ShortcutHint{
				Label:   definition.label,
				KeyName: binding.KeyName,
			})
			break
		}
	}
	return hints
}

const (
	mobileStatusReady                = "ready"
	mobileStatusNotStarted           = "not_started"
	mobileStatusInitializationFailed = "initialization_failed"
	mobileStatusPortUnavailable      = "port_unavailable"
	mobileStatusStopped              = "stopped"
)

func (a *App) mobileBridgeForUse() *mobile.Bridge {
	a.mobileMu.RLock()
	defer a.mobileMu.RUnlock()
	return a.mobileBridge
}

// mobileOutputEnabled 判断当前是否打通了 Mobile 链路（Server 在监听且 bridge 已挂载）。
func (a *App) mobileOutputEnabled() bool {
	a.mobileMu.RLock()
	bridge := a.mobileBridge
	server := a.mobileServer
	a.mobileMu.RUnlock()
	return bridge != nil && server != nil && server.Started()
}

// mobileOutputMode 返回当前配置的 OutputMode（缺省 desktop，配置无效时兜底 desktop）。
func (a *App) mobileOutputMode() string {
	if a.configManager == nil {
		return "desktop"
	}
	mode := a.configManager.Get().OutputMode
	if mode == "" {
		return "desktop"
	}
	return mode
}

// routeSolutionEvent 对解题输出事件执行 OutputMode Router（约束一）。
// 本方法支撑函数 routeDecision 提供决策（decideRoute），保证可单测。
func (a *App) routeSolutionEvent(event string, data ...interface{}) {
	decision := decideRoute(a.mobileOutputMode(), a.mobileOutputEnabled())

	if decision.needMobile {
		if bridge := a.mobileBridgeForUse(); bridge != nil {
			bridge.Publish(event, data...)
		}
	}

	if decision.needDesktop {
		runtimeEmit(a.ctx, event, data...)
	} else if event == "solution" || event == "solution-error" {
		// mobile 模式：桌面不渲染答案，但终态告知前端「已转发到手机端」，收敛 loading 状态
		runtimeEmit(a.ctx, "solution-hidden", map[string]bool{"success": event == "solution"})
	}
}

// setupMobile 启动 Mobile 链路（失败仅降级，绝不影响主功能）。
// Bridge 和配置订阅在 App 生命周期内只创建一次；Server 可在失败后重新监听。
func (a *App) setupMobile() {
	a.mobileMu.Lock()
	defer a.mobileMu.Unlock()
	a.startMobileLocked()
}

func (a *App) startMobileLocked() {
	if a.mobileStatus == "" || a.mobileStatus == mobileStatusStopped {
		a.mobileStatus = mobileStatusNotStarted
	}

	if a.mobileBridge == nil {
		a.mobileBridge = mobile.NewBridge()
	}
	bridge := a.mobileBridge
	if a.configManager != nil {
		cfg := a.configManager.Get()
		bridge.SetModel(cfg.Model)
		bridge.SetHints(mobileShortcutHints(cfg.Shortcuts))
		if !a.mobileConfigSubscribed {
			a.configManager.Subscribe(func(newConfig config.Config, _ config.Config) {
				bridge.SetHints(mobileShortcutHints(newConfig.Shortcuts))
			})
			a.mobileConfigSubscribed = true
		}
	}

	if a.mobileServer == nil {
		server, err := mobile.NewServer(bridge)
		if err != nil {
			a.mobileStatus = mobileStatusInitializationFailed
			logger.Printf("[mobile] 创建 Mobile 服务失败（已降级，不影响主功能）: %v", err)
			return
		}
		a.mobileServer = server
	}

	if a.mobileServer.Started() {
		a.mobileStatus = mobileStatusReady
		return
	}
	if err := a.mobileServer.Start(); err != nil {
		if errors.Is(err, mobile.ErrPortUnavailable) {
			a.mobileStatus = mobileStatusPortUnavailable
		} else {
			a.mobileStatus = mobileStatusInitializationFailed
		}
		logger.Printf("[mobile] 启动 Mobile 服务失败（已降级）: %v", err)
		return
	}
	a.mobileStatus = mobileStatusReady
	logger.Printf("[mobile] Q-Solver 局域网同步端就绪: %s", a.mobileServer.SafeURL())
}

// shutdownMobile 关闭 Mobile 链路（App.OnShutdown 调用）。
func (a *App) shutdownMobile() {
	a.mobileMu.Lock()
	defer a.mobileMu.Unlock()
	if a.mobileServer != nil {
		a.mobileServer.Stop()
	}
	a.mobileStatus = mobileStatusStopped
}

// routeDecision 描述一次解题输出事件的路由决策结果。
type routeDecision struct {
	needDesktop bool // 桌面 Wails 需渲染
	needMobile  bool // Mobile Bridge 需推送
}

// decideRoute 依据 OutputMode 与 Server 可用性计算路由。纯函数，可单测。
//
//	desktop           → 仅桌面
//	mobile +server    → 仅手机（mobile）
//	mobile + 不可用    → 降级桌面（约束二，避免丢答案）
//	dual              → 桌面 + 手机
func decideRoute(mode string, serverAvailable bool) routeDecision {
	switch mode {
	case "mobile":
		if serverAvailable {
			return routeDecision{needDesktop: false, needMobile: true}
		}
		return routeDecision{needDesktop: true, needMobile: false} // 降级
	case "dual":
		return routeDecision{needDesktop: true, needMobile: true}
	default: // desktop / 未知 mode，一律桌面
		return routeDecision{needDesktop: true, needMobile: false}
	}
}

// decideScrollRoute 计算一次滚动控制的目标。
// 与解题输出不同，一次性控制在 Mobile 不可用时不进入孤立 Bridge，而是安全回退到桌面。
func decideScrollRoute(mode string, serverAvailable bool) routeDecision {
	if !serverAvailable {
		return routeDecision{needDesktop: true, needMobile: false}
	}
	switch mode {
	case "mobile":
		return routeDecision{needDesktop: false, needMobile: true}
	case "dual":
		return routeDecision{needDesktop: true, needMobile: true}
	default:
		return routeDecision{needDesktop: true, needMobile: false}
	}
}

// MobileInfo 连接移动信息（供 settings 面板展示；仅就绪时包含连接 URL）。
type MobileInfo struct {
	Available bool   `json:"available"`
	URL       string `json:"url"`
	Port      int    `json:"port"`
	Status    string `json:"status"`
	Retryable bool   `json:"retryable"`
}

func (a *App) mobileInfoLocked() MobileInfo {
	status := a.mobileStatus
	if status == "" {
		status = mobileStatusNotStarted
	}
	if a.mobileBridge != nil && a.mobileServer != nil && a.mobileServer.Started() {
		return MobileInfo{
			Available: true,
			URL:       a.mobileServer.URL(),
			Port:      a.mobileServer.Port(),
			Status:    mobileStatusReady,
		}
	}
	retryable := status == mobileStatusNotStarted ||
		status == mobileStatusInitializationFailed ||
		status == mobileStatusPortUnavailable
	return MobileInfo{Status: status, Retryable: retryable}
}

// GetMobileOutputInfo 返回移动端连接入口信息。
func (a *App) GetMobileOutputInfo() MobileInfo {
	a.mobileMu.RLock()
	defer a.mobileMu.RUnlock()
	return a.mobileInfoLocked()
}

// RetryMobileOutput 重试附加的 Mobile 服务。已运行时幂等返回，不触发窗口隐藏策略。
func (a *App) RetryMobileOutput() MobileInfo {
	a.mobileMu.Lock()
	defer a.mobileMu.Unlock()
	a.startMobileLocked()
	return a.mobileInfoLocked()
}
