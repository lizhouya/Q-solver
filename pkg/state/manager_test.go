package state

import "testing"

func TestToggleClickThroughWithoutWindowHandleReturnsError(t *testing.T) {
	sm := NewStateManager()
	var events []string
	sm.emitEvent = func(name string, data ...interface{}) {
		events = append(events, name)
	}

	enabled, err := sm.ToggleClickThrough()
	if err == nil {
		t.Fatal("窗口句柄未初始化时应返回错误")
	}
	if enabled {
		t.Fatal("切换失败时应返回未改变的 false 状态")
	}
	if sm.IsClickThrough() {
		t.Fatal("切换失败时不得改变 ClickThrough 状态")
	}
	if len(events) != 0 {
		t.Fatalf("切换失败时不得发送成功状态事件，实际 %v", events)
	}
}
