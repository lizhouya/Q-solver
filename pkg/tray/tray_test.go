package tray

import (
	"errors"
	"sync"
	"testing"
)

// mockController 模拟平台托盘控制器，验证管理器生命周期而不依赖真实 Win32。
type mockController struct {
	mu           sync.Mutex
	startCalls   int
	stopCalls    int
	startErr     error
	stopErr      error
	lastHandlers TrayEventHandlers
}

func (m *mockController) Start(handlers TrayEventHandlers) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls++
	m.lastHandlers = handlers
	return m.startErr
}

func (m *mockController) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalls++
	return m.stopErr
}

// counts 返回 start/stop 调用次数（测试断言用）。
func (m *mockController) counts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls, m.stopCalls
}

// reset 清零调用记录，用于幂等性测试。
func (m *mockController) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls = 0
	m.stopCalls = 0
	m.startErr = nil
	m.stopErr = nil
}

// newManager 构造测试管理器（注入 mock 控制器）。
func newManager(m *mockController) *TrayManager {
	return &TrayManager{
		controller: m,
		handlers:   TrayEventHandlers{},
	}
}

// --- 初始状态 ---

func TestInitialNotRunning(t *testing.T) {
	m := newManager(&mockController{})
	if m.IsRunning() {
		t.Fatal("初始状态应为未运行")
	}
}

// --- Start ---

func TestStartDelegatesToController(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	if err := m.Start(); err != nil {
		t.Fatalf("Start 不应出错: %v", err)
	}
	starts, stops := mock.counts()
	if starts != 1 {
		t.Fatalf("Start 应调用底层 1 次，得到 %d", starts)
	}
	if stops != 0 {
		t.Fatalf("Start 不应调用底层 Stop，得到 %d", stops)
	}
	if !m.IsRunning() {
		t.Fatal("Start 成功后应处于运行中")
	}
}

func TestStartPropagatesHandlers(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	called := false
	m.handlers.OnShow = func() { called = true }

	if err := m.Start(); err != nil {
		t.Fatalf("Start 不应出错: %v", err)
	}
	if mock.lastHandlers.OnShow == nil {
		t.Fatal("底层应收到 OnShow 回调")
	}
	// 触发传递的回调，验证链路畅通
	mock.lastHandlers.OnShow()
	if !called {
		t.Fatal("通过回调应能触发上层动作")
	}
}

func TestStartIdempotent(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	_ = m.Start()
	mock.reset()
	if err := m.Start(); err != nil {
		t.Fatalf("重复 Start 不应出错: %v", err)
	}
	starts, _ := mock.counts()
	if starts != 0 {
		t.Fatalf("已运行时重复 Start 不应调用底层（幂等），得到 %d", starts)
	}
	if !m.IsRunning() {
		t.Fatal("重复 Start 后应保持运行中")
	}
}

func TestStartControllerError(t *testing.T) {
	mock := &mockController{startErr: errors.New("mock start error")}
	m := newManager(mock)

	if err := m.Start(); err == nil {
		t.Fatal("底层失败时 Start 应返回错误")
	}
	if m.IsRunning() {
		t.Fatal("底层失败时不应标记为运行中")
	}
}

// --- Stop ---

func TestStopDelegatesToController(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	_ = m.Start()
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop 不应出错: %v", err)
	}
	_, stops := mock.counts()
	if stops != 1 {
		t.Fatalf("Stop 应调用底层 1 次，得到 %d", stops)
	}
	if m.IsRunning() {
		t.Fatal("Stop 后应不再运行")
	}
}

func TestStopIdempotent(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	_ = m.Start()
	_ = m.Stop()
	mock.reset()
	if err := m.Stop(); err != nil {
		t.Fatalf("重复 Stop 不应出错: %v", err)
	}
	_, stops := mock.counts()
	if stops != 0 {
		t.Fatalf("已停止时重复 Stop 不应调用底层（实际 %d）", stops)
	}
}

func TestStopBeforeStartNoOp(t *testing.T) {
	mock := &mockController{}
	m := newManager(mock)

	if err := m.Stop(); err != nil {
		t.Fatalf("未启动时 Stop 不应出错: %v", err)
	}
	_, stops := mock.counts()
	if stops != 0 {
		t.Fatalf("未启动时 Stop 不应调用底层")
	}
	if m.IsRunning() {
		t.Fatal("未启动时 Stop 后应保持未运行")
	}
}

// --- 停止失败：状态复位允许重试 ---

func TestStopControllerErrorTracksState(t *testing.T) {
	mock := &mockController{stopErr: errors.New("mock stop error")}
	m := newManager(mock)

	_ = m.Start()
	if err := m.Stop(); err == nil {
		t.Fatal("底层 Stop 失败时应返回错误")
	}
	if m.IsRunning() {
		t.Fatal("Stop 失败后也应标记为未运行（允许重试）")
	}
}
