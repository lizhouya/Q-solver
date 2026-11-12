//go:build windows

package shortcut

import "testing"

func TestWindowsFunctionKeysDispatchFastAndDelete(t *testing.T) {
	d := &mockDelegate{}
	s := newTestService(d)
	s.manager.Shortcuts = map[string]KeyBinding{
		"send_fast": {ComboID: "121", KeyName: "F10"},
		"delete":    {ComboID: "123", KeyName: "F12"},
	}
	s.manager.OnTrigger = s.handleTrigger

	previous := globalManager
	globalManager = s.manager
	t.Cleanup(func() { globalManager = previous })

	s.manager.heldKeys = map[uint32]bool{121: true}
	if !onKeysChanged() {
		t.Fatal("F10 应匹配并吞掉 send_fast 快捷键")
	}
	s.manager.heldKeys = map[uint32]bool{123: true}
	if !onKeysChanged() {
		t.Fatal("F12 应匹配并吞掉 delete 快捷键")
	}

	if d.quickSendCalls != 1 || d.deleteScreenshotCalls != 1 {
		t.Fatalf("F10/F12 分发错误: quick=%d delete=%d", d.quickSendCalls, d.deleteScreenshotCalls)
	}
}
