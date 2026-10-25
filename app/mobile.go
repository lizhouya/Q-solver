package app

import (
	"Q-Solver/pkg/answersummary"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/mobile"
	"Q-Solver/pkg/shortcut"
	"errors"
)

type DesktopSolveEvent struct {
	AnswerItems    []answersummary.Item `json:"answerItems,omitempty"`
	AnswerCopyText string               `json:"answerCopyText,omitempty"`
	Epoch          uint64               `json:"epoch"`
	SessionID      string               `json:"sessionId"`
	RecordID       string               `json:"recordId"`
	Event          string               `json:"event"`
	Text           string               `json:"text,omitempty"`
	Images         []string             `json:"images,omitempty"`
	Success        *bool                `json:"success,omitempty"`
}

type desktopSolveTarget struct {
	epoch     uint64
	sessionID string
	recordID  string
}

// 解题输出事件白名单：仅这些事件受 OutputMode 路由控制（02-plan §2.1 约束一）。
// 其余所有事件（toast/screenshot/start-solving/user-message/shortcut/…）不经过路由，永远走桌面 Wails。
type mobileStateManager interface {
	LoadOrCreate() (config.MobileRuntimeState, error)
	AdvanceClearSeq() (config.MobileRuntimeState, error)
	ResetLink() (config.MobileRuntimeState, error)
	SetStableLink(bool) (config.MobileRuntimeState, error)
}

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
	{label: "清空同步端", actions: []string{"clear_mobile_display"}},
	{label: "开始下一套", actions: []string{"start_next_exam"}},
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
	mobileStatusStateUnavailable     = "mobile_state_unavailable"
	mobileStatusStopped              = "stopped"
)

func (a *App) mobileBridgeForUse() *mobile.Bridge {
	a.mobileMu.RLock()
	defer a.mobileMu.RUnlock()
	return a.mobileBridge
}

func (a *App) mobileIsClosing() bool {
	a.mobileMu.RLock()
	defer a.mobileMu.RUnlock()
	return a.mobileClosing
}

func (a *App) mobileDisplayEpoch() uint64 {
	if bridge := a.mobileBridgeForUse(); bridge != nil {
		return bridge.CurrentClearSeq()
	}
	return 0
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

// routeTaskSolutionEvent 仅允许任务写入启动时捕获的 Mobile 显示代际。
// 桌面路由不受清屏影响；题库捕获在调用本方法前已经完成。
func (a *App) routeTaskSolutionEvent(expected uint64, event string, data ...interface{}) {
	decision := decideRoute(a.mobileOutputMode(), a.mobileOutputEnabled())

	if decision.needMobile {
		if bridge := a.mobileBridgeForUse(); bridge != nil {
			bridge.PublishIfCurrent(expected, event, data...)
		}
	}

	if decision.needDesktop {
		runtimeEmit(a.ctx, event, data...)
	} else if event == "solution" || event == "solution-error" {
		runtimeEmit(a.ctx, "solution-hidden", map[string]bool{"success": event == "solution"})
	}
}

func (a *App) routeConversationSolutionEvent(expectedDisplay uint64, target desktopSolveTarget, event string, data ...interface{}) {
	decision := decideRoute(a.mobileOutputMode(), a.mobileOutputEnabled())
	if decision.needMobile {
		if bridge := a.mobileBridgeForUse(); bridge != nil {
			bridge.PublishIfCurrent(expectedDisplay, event, data...)
		}
	}

	envelope := DesktopSolveEvent{
		Epoch:     target.epoch,
		SessionID: target.sessionID,
		RecordID:  target.recordID,
		Event:     event,
	}
	if len(data) > 0 {
		envelope.Text, _ = data[0].(string)
	}
	if event == "solution" && len(data) > 1 {
		if result, ok := data[1].(answersummary.Result); ok {
			envelope.AnswerItems = append([]answersummary.Item(nil), result.Items...)
			envelope.AnswerCopyText, _ = answersummary.BuildCopyText(result.Items)
		}
	}
	if decision.needDesktop {
		runtimeEmit(a.ctx, "desktop-solve-event", envelope)
	} else if event == "solution" || event == "solution-error" {
		success := event == "solution"
		envelope.Event = "hidden"
		envelope.Success = &success
		runtimeEmit(a.ctx, "desktop-solve-event", envelope)
	}
}

// setupMobile 启动 Mobile 链路（失败仅降级，绝不影响主功能）。
func (a *App) setupMobile() {
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	if a.mobileIsClosing() {
		return
	}
	a.startMobileUnderOp()
}

func (a *App) setMobileStatus(status string) {
	a.mobileMu.Lock()
	a.mobileStatus = status
	a.mobileMu.Unlock()
}

func (a *App) startMobileUnderOp() {
	a.setMobileStatus(mobileStatusNotStarted)

	a.mobileMu.RLock()
	stateReady := a.mobileStateReady
	state := a.mobileRuntimeState
	bridge := a.mobileBridge
	server := a.mobileServer
	a.mobileMu.RUnlock()

	if !stateReady {
		if a.mobileStateManager == nil {
			a.mobileStateManager = config.NewMobileStateManager()
		}
		loaded, err := a.mobileStateManager.LoadOrCreate()
		if err != nil {
			a.setMobileStatus(mobileStatusStateUnavailable)
			logger.Println("[mobile] Mobile 专用状态不可用（已降级，不影响主功能）")
			return
		}
		state = loaded
		a.mobileMu.Lock()
		a.mobileRuntimeState = loaded
		a.mobileStateReady = true
		a.mobileMu.Unlock()
	}

	if bridge == nil {
		created, err := mobile.NewBridge(state.ClearSeq)
		if err != nil {
			a.setMobileStatus(mobileStatusInitializationFailed)
			logger.Println("[mobile] 创建 Mobile Bridge 失败（已降级，不影响主功能）")
			return
		}
		bridge = created
		if a.configManager != nil {
			cfg := a.configManager.Get()
			bridge.SetModel(cfg.Model)
			bridge.SetHints(mobileShortcutHints(cfg.Shortcuts))
		}
		a.mobileMu.Lock()
		a.mobileBridge = bridge
		a.mobileMu.Unlock()
	}

	if a.configManager != nil {
		a.mobileMu.Lock()
		shouldSubscribe := !a.mobileConfigSubscribed
		if shouldSubscribe {
			a.mobileConfigSubscribed = true
		}
		a.mobileMu.Unlock()
		if shouldSubscribe {
			a.configManager.Subscribe(func(newConfig config.Config, _ config.Config) {
				if current := a.mobileBridgeForUse(); current != nil {
					current.SetHints(mobileShortcutHints(newConfig.Shortcuts))
				}
			})
		}
	}

	if server == nil {
		created, err := mobile.NewServer(bridge, state.Token)
		if err != nil {
			a.setMobileStatus(mobileStatusInitializationFailed)
			logger.Println("[mobile] 创建 Mobile 服务失败（已降级，不影响主功能）")
			return
		}
		server = created
		a.mobileMu.Lock()
		a.mobileServer = server
		a.mobileMu.Unlock()
	}

	if server.Started() {
		a.setMobileStatus(mobileStatusReady)
		return
	}
	if err := server.Start(); err != nil {
		if errors.Is(err, mobile.ErrPortUnavailable) {
			a.setMobileStatus(mobileStatusPortUnavailable)
		} else {
			a.setMobileStatus(mobileStatusInitializationFailed)
		}
		logger.Println("[mobile] 启动 Mobile 服务失败（已降级）")
		return
	}
	a.setMobileStatus(mobileStatusReady)
	logger.Printf("[mobile] Q-Solver 同步端就绪: %s", server.SafeURL())
}

// shutdownMobile 设置关闭栅栏后摘出运行对象，再在不持有 App 状态锁时关闭连接。
func (a *App) shutdownMobile() {
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	a.mobileMu.Lock()
	if a.mobileClosing {
		a.mobileMu.Unlock()
		return
	}
	a.mobileClosing = true
	bridge := a.mobileBridge
	server := a.mobileServer
	a.mobileBridge = nil
	a.mobileServer = nil
	a.mobileStatus = mobileStatusStopped
	a.mobileMu.Unlock()

	if bridge != nil {
		bridge.Close()
	}
	if server != nil {
		server.Stop()
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

// MobileInfo 是供桌面前端展示的非敏感 Mobile 状态。
type MobileInfo struct {
	Available         bool   `json:"available"`
	URL               string `json:"url"`
	Port              int    `json:"port"`
	Status            string `json:"status"`
	Retryable         bool   `json:"retryable"`
	StableLinkEnabled bool   `json:"stableLinkEnabled"`
	CanClear          bool   `json:"canClear"`
	CanResetLink      bool   `json:"canResetLink"`
}

type MobileCommandResult struct {
	Success bool       `json:"success"`
	Code    string     `json:"code"`
	Info    MobileInfo `json:"info"`
}

func (a *App) mobileInfoLocked() MobileInfo {
	status := a.mobileStatus
	if status == "" {
		status = mobileStatusNotStarted
	}
	info := MobileInfo{
		Status:            status,
		StableLinkEnabled: a.mobileRuntimeState.StableLink,
		CanClear:          a.mobileStateReady && a.mobileBridge != nil && !a.mobileClosing,
		CanResetLink:      a.mobileStateReady && a.mobileBridge != nil && !a.mobileClosing,
	}
	if a.mobileBridge != nil && a.mobileServer != nil && a.mobileServer.Started() {
		info.Available = true
		info.URL = a.mobileServer.URL()
		info.Port = a.mobileServer.Port()
		info.Status = mobileStatusReady
		return info
	}
	info.Retryable = status == mobileStatusNotStarted ||
		status == mobileStatusInitializationFailed ||
		status == mobileStatusPortUnavailable ||
		status == mobileStatusStateUnavailable
	return info
}

func (a *App) mobileInfo() MobileInfo {
	a.mobileMu.RLock()
	defer a.mobileMu.RUnlock()
	return a.mobileInfoLocked()
}

func (a *App) GetMobileOutputInfo() MobileInfo {
	return a.mobileInfo()
}

// RetryMobileOutput 只重新建立运行链路，不轮换 Token 或推进清屏边界。
func (a *App) RetryMobileOutput() MobileInfo {
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	if a.mobileIsClosing() {
		return a.mobileInfo()
	}
	a.startMobileUnderOp()
	return a.mobileInfo()
}

func (a *App) mobileCommandResult(success bool, code string) MobileCommandResult {
	return MobileCommandResult{Success: success, Code: code, Info: a.mobileInfo()}
}

func (a *App) mobileMutationRuntimeUnderOp() (*mobile.Bridge, *mobile.Server, MobileCommandResult, bool) {
	a.mobileMu.RLock()
	closing := a.mobileClosing
	ready := a.mobileStateReady
	bridge := a.mobileBridge
	server := a.mobileServer
	a.mobileMu.RUnlock()
	if closing {
		return nil, nil, a.mobileCommandResult(false, "mobile_shutting_down"), false
	}
	if !ready || a.mobileStateManager == nil {
		return nil, nil, a.mobileCommandResult(false, "mobile_state_unavailable"), false
	}
	if bridge == nil {
		return nil, server, a.mobileCommandResult(false, "mobile_runtime_unavailable"), false
	}
	return bridge, server, MobileCommandResult{}, true
}

func (a *App) applyMobileRuntimeState(state config.MobileRuntimeState) {
	a.mobileMu.Lock()
	a.mobileRuntimeState = state
	a.mobileStateReady = true
	a.mobileMu.Unlock()
}

func (a *App) ClearMobileDisplay(confirmed bool) MobileCommandResult {
	if !confirmed {
		return a.mobileCommandResult(false, "confirmation_required")
	}
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	bridge, _, failed, ok := a.mobileMutationRuntimeUnderOp()
	if !ok {
		return failed
	}
	state, err := a.mobileStateManager.AdvanceClearSeq()
	if err != nil {
		return a.mobileCommandResult(false, "mobile_state_persist_failed")
	}
	a.applyMobileRuntimeState(state)
	bridge.ClearDisplay(state.ClearSeq)
	return a.mobileCommandResult(true, "cleared")
}

func (a *App) ResetMobileOutputLink(confirmed bool) MobileCommandResult {
	if !confirmed {
		return a.mobileCommandResult(false, "confirmation_required")
	}
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	bridge, server, failed, ok := a.mobileMutationRuntimeUnderOp()
	if !ok {
		return failed
	}
	state, err := a.mobileStateManager.ResetLink()
	if err != nil {
		return a.mobileCommandResult(false, "mobile_state_persist_failed")
	}
	a.applyMobileRuntimeState(state)
	bridge.ClearDisplay(state.ClearSeq)
	if server != nil {
		server.RotateToken(state.Token)
	}
	return a.mobileCommandResult(true, "link_reset")
}

func (a *App) SetMobileStableLink(enabled, confirmed bool) MobileCommandResult {
	if !confirmed {
		return a.mobileCommandResult(false, "confirmation_required")
	}
	a.mobileOpMu.Lock()
	defer a.mobileOpMu.Unlock()
	bridge, server, failed, ok := a.mobileMutationRuntimeUnderOp()
	if !ok {
		return failed
	}
	a.mobileMu.RLock()
	alreadyCurrent := a.mobileRuntimeState.StableLink == enabled
	a.mobileMu.RUnlock()
	if alreadyCurrent {
		return a.mobileCommandResult(true, "already_current")
	}
	state, err := a.mobileStateManager.SetStableLink(enabled)
	if err != nil {
		return a.mobileCommandResult(false, "mobile_state_persist_failed")
	}
	a.applyMobileRuntimeState(state)
	bridge.ClearDisplay(state.ClearSeq)
	if server != nil {
		server.RotateToken(state.Token)
	}
	code := "stable_link_disabled"
	if enabled {
		code = "stable_link_enabled"
	}
	return a.mobileCommandResult(true, code)
}
