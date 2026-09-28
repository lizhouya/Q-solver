package app

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/mobile"
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

// mobileOutputEnabled 判断当前是否打通了 Mobile 链路（Server 在监听且 bridge 已挂载）。
func (a *App) mobileOutputEnabled() bool {
	return a.mobileBridge != nil && a.mobileServer != nil && a.mobileServer.Started()
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

	if decision.needMobile && a.mobileBridge != nil {
		a.mobileBridge.Publish(event, data...)
	}

	if decision.needDesktop {
		runtimeEmit(a.ctx, event, data...)
	} else if event == "solution" || event == "solution-error" {
		// mobile 模式：桌面不渲染答案，但终态告知前端「已转发到手机端」，收敛 loading 状态
		runtimeEmit(a.ctx, "solution-hidden", map[string]bool{"success": event == "solution"})
	}
}

// setupMobile 启动 Mobile 链路（约束二：失败仅降级，绝不影响主功能）。
// 在 App.Startup 中调用。
func (a *App) setupMobile() {
	bridge := mobile.NewBridge()
	if a.configManager != nil {
		bridge.SetModel(a.configManager.Get().Model)
	}
	a.mobileBridge = bridge

	srv, err := mobile.NewServer(bridge)
	if err != nil {
		logger.Printf("[mobile] 创建 Mobile 服务失败（已降级，不影响主功能）: %v", err)
		return
	}
	if err := srv.Start(); err != nil {
		logger.Printf("[mobile] 启动 Mobile 服务失败（已降级）: %v", err)
		return
	}
	a.mobileServer = srv
	logger.Printf("[mobile] Q-Solver 局域网同步端就绪: %s", srv.SafeURL())
}

// shutdownMobile 关闭 Mobile 链路（App.OnShutdown 调用）。
func (a *App) shutdownMobile() {
	if a.mobileServer != nil {
		a.mobileServer.Stop()
	}
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

// MobileInfo 连接移动信息（导入 settings 面板展示用；不含真实 token）。
type MobileInfo struct {
	Available bool   `json:"available"`
	URL       string `json:"url"`
	Port      int    `json:"port"`
}

// GetMobileOutputInfo 返回移动端连接入口信息（Wails binding，供前端面板展示/复制）。
func (a *App) GetMobileOutputInfo() MobileInfo {
	if !a.mobileOutputEnabled() {
		return MobileInfo{Available: false, Port: 0}
	}
	return MobileInfo{
		Available: true,
		URL:       a.mobileServer.URL(),
	}
}