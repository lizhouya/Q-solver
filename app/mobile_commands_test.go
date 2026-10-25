package app

import (
	"errors"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/mobile"
)

const appTestMobileToken = "0123456789abcdef0123456789abcdef"
const appTestRotatedMobileToken = "fedcba9876543210fedcba9876543210"

type fakeMobileStateManager struct {
	state          config.MobileRuntimeState
	loadErr        error
	advanceErr     error
	resetErr       error
	stableErr      error
	loadCalls      int
	advanceCalls   int
	resetCalls     int
	setStableCalls int
}

func (f *fakeMobileStateManager) LoadOrCreate() (config.MobileRuntimeState, error) {
	f.loadCalls++
	return f.state, f.loadErr
}

func (f *fakeMobileStateManager) AdvanceClearSeq() (config.MobileRuntimeState, error) {
	f.advanceCalls++
	if f.advanceErr != nil {
		return config.MobileRuntimeState{}, f.advanceErr
	}
	f.state.ClearSeq++
	f.state.Generation++
	return f.state, nil
}

func (f *fakeMobileStateManager) ResetLink() (config.MobileRuntimeState, error) {
	f.resetCalls++
	if f.resetErr != nil {
		return config.MobileRuntimeState{}, f.resetErr
	}
	f.state.ClearSeq++
	f.state.Generation++
	f.rotateToken()
	return f.state, nil
}

func (f *fakeMobileStateManager) SetStableLink(enabled bool) (config.MobileRuntimeState, error) {
	f.setStableCalls++
	if f.stableErr != nil {
		return config.MobileRuntimeState{}, f.stableErr
	}
	if f.state.StableLink == enabled {
		return f.state, nil
	}
	f.state.StableLink = enabled
	f.state.ClearSeq++
	f.state.Generation++
	f.rotateToken()
	return f.state, nil
}

func (f *fakeMobileStateManager) rotateToken() {
	if f.state.Token == appTestMobileToken {
		f.state.Token = appTestRotatedMobileToken
	} else {
		f.state.Token = appTestMobileToken
	}
}

func newMobileCommandTestApp(t *testing.T, manager *fakeMobileStateManager) *App {
	t.Helper()
	bridge, err := mobile.NewBridge(manager.state.ClearSeq)
	if err != nil {
		t.Fatal("创建测试 Bridge 失败")
	}
	server, err := mobile.NewServer(bridge, manager.state.Token)
	if err != nil {
		t.Fatal("创建测试 Server 失败")
	}
	return &App{
		mobileBridge:       bridge,
		mobileServer:       server,
		mobileStateManager: manager,
		mobileRuntimeState: manager.state,
		mobileStateReady:   true,
		mobileStatus:       mobileStatusNotStarted,
	}
}

func mobileManagerCallCount(manager *fakeMobileStateManager) int {
	return manager.loadCalls + manager.advanceCalls + manager.resetCalls + manager.setStableCalls
}

func TestMobileCommandsRequireConfirmation(t *testing.T) {
	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		Generation: 1,
	}}
	a := newMobileCommandTestApp(t, manager)
	before := a.mobileBridge.GetState()

	for _, result := range []MobileCommandResult{
		a.ClearMobileDisplay(false),
		a.ResetMobileOutputLink(false),
		a.SetMobileStableLink(false, false),
	} {
		if result.Success || result.Code != "confirmation_required" {
			t.Fatalf("未确认操作应被拒绝，实际 success=%v code=%q", result.Success, result.Code)
		}
	}
	if mobileManagerCallCount(manager) != 0 {
		t.Fatal("未确认操作不得调用 Mobile 状态管理器")
	}
	after := a.mobileBridge.GetState()
	if after.ClearSeq != before.ClearSeq || after.Seq != before.Seq {
		t.Fatal("未确认操作不得推进显示边界")
	}
}

func TestClearMobileDisplayCommitsBeforeClearingRuntime(t *testing.T) {
	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		ClearSeq:   4,
		Generation: 7,
	}}
	a := newMobileCommandTestApp(t, manager)
	a.mobileBridge.Publish("solution-stream-start")
	a.mobileBridge.Publish("solution-stream-chunk", "待清理答案")

	result := a.ClearMobileDisplay(true)
	if !result.Success || result.Code != "cleared" || manager.advanceCalls != 1 {
		t.Fatalf("清屏结果不正确: success=%v code=%q calls=%d", result.Success, result.Code, manager.advanceCalls)
	}
	state := a.mobileBridge.GetState()
	if state.ClearSeq != 5 || state.Status != mobile.StatusIdle || state.Payload.Answer != "" {
		t.Fatalf("运行态未切换到已持久化的清屏边界: clear=%d status=%q", state.ClearSeq, state.Status)
	}
	if a.mobileRuntimeState != manager.state || a.mobileRuntimeState.Token != appTestMobileToken {
		t.Fatal("清屏不得轮换链接，App 状态应与 manager 一致")
	}
}

func TestMobilePersistenceFailureLeavesRuntimeUnchanged(t *testing.T) {
	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		ClearSeq:   2,
		Generation: 3,
	}, advanceErr: errors.New("synthetic persistence failure")}
	a := newMobileCommandTestApp(t, manager)
	a.mobileBridge.Publish("solution-stream-start")
	a.mobileBridge.Publish("solution-stream-chunk", "保留答案")
	before := a.mobileBridge.GetState()
	beforeRuntime := a.mobileRuntimeState

	result := a.ClearMobileDisplay(true)
	if result.Success || result.Code != "mobile_state_persist_failed" {
		t.Fatalf("持久化失败应返回稳定错误码，实际 success=%v code=%q", result.Success, result.Code)
	}
	after := a.mobileBridge.GetState()
	if after.ClearSeq != before.ClearSeq || after.Seq != before.Seq || after.Payload.Answer != before.Payload.Answer {
		t.Fatal("持久化失败后 Bridge 状态发生变化")
	}
	if a.mobileRuntimeState != beforeRuntime {
		t.Fatal("持久化失败后 App 运行状态发生变化")
	}
}

func TestResetAndStableModeWorkWithoutStartingServer(t *testing.T) {
	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		ClearSeq:   10,
		Generation: 2,
	}}
	a := newMobileCommandTestApp(t, manager)
	if a.mobileServer.Started() {
		t.Fatal("测试前提：Server 不应监听")
	}

	reset := a.ResetMobileOutputLink(true)
	if !reset.Success || reset.Code != "link_reset" || manager.resetCalls != 1 {
		t.Fatalf("重置链接失败: success=%v code=%q", reset.Success, reset.Code)
	}
	if a.mobileServer.Started() || a.mobileBridge.CurrentClearSeq() != 11 {
		t.Fatal("重置不得隐式启动服务，且必须推进清屏边界")
	}

	disable := a.SetMobileStableLink(false, true)
	if !disable.Success || disable.Code != "stable_link_disabled" || manager.setStableCalls != 1 {
		t.Fatalf("关闭固定链接失败: success=%v code=%q", disable.Success, disable.Code)
	}
	if a.mobileServer.Started() || a.mobileBridge.CurrentClearSeq() != 12 {
		t.Fatal("模式切换不得隐式启动服务，且必须推进清屏边界")
	}

	same := a.SetMobileStableLink(false, true)
	if !same.Success || same.Code != "already_current" || manager.setStableCalls != 1 {
		t.Fatal("相同模式请求应幂等且不得调用 manager")
	}
}

func TestShutdownFencesLateMobileOperations(t *testing.T) {
	manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
		StableLink: true,
		Token:      appTestMobileToken,
		Generation: 1,
	}}
	a := newMobileCommandTestApp(t, manager)
	a.shutdownMobile()
	beforeCalls := mobileManagerCallCount(manager)

	for _, result := range []MobileCommandResult{
		a.ClearMobileDisplay(true),
		a.ResetMobileOutputLink(true),
		a.SetMobileStableLink(false, true),
	} {
		if result.Success || result.Code != "mobile_shutting_down" {
			t.Fatalf("关闭后的操作应被栅栏拒绝，实际 success=%v code=%q", result.Success, result.Code)
		}
	}
	info := a.RetryMobileOutput()
	if info.Available || info.Status != mobileStatusStopped {
		t.Fatalf("关闭后 Retry 不得重启服务，实际 available=%v status=%q", info.Available, info.Status)
	}
	if mobileManagerCallCount(manager) != beforeCalls {
		t.Fatal("关闭后的操作不得调用状态管理器")
	}
}

func TestResetAndStablePersistenceFailuresLeaveRuntimeUnchanged(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*fakeMobileStateManager)
		invoke func(*App) MobileCommandResult
	}{
		{
			name: "reset",
			setup: func(manager *fakeMobileStateManager) {
				manager.resetErr = errors.New("synthetic reset failure")
			},
			invoke: func(a *App) MobileCommandResult { return a.ResetMobileOutputLink(true) },
		},
		{
			name: "stable mode",
			setup: func(manager *fakeMobileStateManager) {
				manager.stableErr = errors.New("synthetic stable failure")
			},
			invoke: func(a *App) MobileCommandResult { return a.SetMobileStableLink(false, true) },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &fakeMobileStateManager{state: config.MobileRuntimeState{
				StableLink: true,
				Token:      appTestMobileToken,
				ClearSeq:   8,
				Generation: 4,
			}}
			test.setup(manager)
			a := newMobileCommandTestApp(t, manager)
			beforeBridge := a.mobileBridge.GetState()
			beforeRuntime := a.mobileRuntimeState

			result := test.invoke(a)
			if result.Success || result.Code != "mobile_state_persist_failed" {
				t.Fatalf("持久化失败应返回稳定错误码，实际 success=%v code=%q", result.Success, result.Code)
			}
			afterBridge := a.mobileBridge.GetState()
			if afterBridge.ClearSeq != beforeBridge.ClearSeq || afterBridge.Seq != beforeBridge.Seq {
				t.Fatal("持久化失败后 Bridge 状态发生变化")
			}
			if a.mobileRuntimeState != beforeRuntime || a.mobileServer.Started() {
				t.Fatal("持久化失败后 App 或 Server 运行态发生变化")
			}
		})
	}
}

func TestTaskMobileEventsRespectCapturedDisplayEpoch(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	configManager := config.NewConfigManager()
	if err := configManager.Load(); err != nil {
		t.Fatal("加载隔离测试配置失败")
	}
	if err := configManager.Patch(func(cfg *config.Config) { cfg.OutputMode = "dual" }); err != nil {
		t.Fatal("写入隔离测试配置失败")
	}

	bridge, err := mobile.NewBridge(21)
	if err != nil {
		t.Fatal("创建测试 Bridge 失败")
	}
	a := &App{configManager: configManager, mobileBridge: bridge}
	sub, _ := bridge.Subscribe()
	defer bridge.Unsubscribe(sub)

	captured := a.mobileDisplayEpoch()
	bridge.ClearDisplay(captured + 1)
	<-sub
	before := bridge.GetState()

	a.routeTaskSolutionEvent(captured, "solution-stream-start")
	a.routeTaskSolutionEvent(captured, "solution-error", "迟到错误")
	if after := bridge.GetState(); after.Seq != before.Seq || after.Status != mobile.StatusIdle || len(sub) != 0 {
		t.Fatal("清屏前任务的迟到事件越过了显示边界")
	}

	current := a.mobileDisplayEpoch()
	a.routeTaskSolutionEvent(current, "solution-stream-start")
	select {
	case snapshot := <-sub:
		if snapshot.ClearSeq != current || snapshot.Status != mobile.StatusWaiting {
			t.Fatalf("新显示代际未正常发布: clear=%d status=%q", snapshot.ClearSeq, snapshot.Status)
		}
	default:
		t.Fatal("清屏后的新任务未发布到 Mobile")
	}
}
