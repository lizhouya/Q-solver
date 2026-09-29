package config

import (
	"encoding/json"
	"runtime"
	"testing"
)

func TestDefaultShortcutsToggleWindow(t *testing.T) {
	cfg := NewDefaultConfig()
	kb, ok := cfg.Shortcuts["toggle_window"]
	if !ok {
		t.Fatal("默认快捷键应包含 toggle_window")
	}

	switch runtime.GOOS {
	case "windows":
		if kb.ComboID != "72+160+162" || kb.KeyName != "Ctrl+Shift+H" {
			t.Fatalf("Windows 默认应为 Ctrl+Shift+H(72+160+162)，实际 %+v", kb)
		}
	case "darwin":
		if kb.ComboID != "Cmd+4" || kb.KeyName != "⌘4" {
			t.Fatalf("macOS 默认应为 ⌘4，实际 %+v", kb)
		}
	default:
		t.Skipf("未覆盖平台 %s", runtime.GOOS)
	}
}

func TestToggleWindowNotAliasingToggle(t *testing.T) {
	// 关键约束：toggle_window ≠ toggle（F9 防录屏隐身），默认键互不冲突
	cfg := NewDefaultConfig()
	if cfg.Shortcuts["toggle_window"].ComboID == cfg.Shortcuts["toggle"].ComboID {
		t.Fatal("toggle_window 与 toggle 不得共用同一 ComboID")
	}
}

func TestShortcutsJSONRoundTrip(t *testing.T) {
	cfg := NewDefaultConfig()
	raw := cfg.ToJSON()

	var back Config
	if err := json.Unmarshal([]byte(raw), &back); err != nil {
		t.Fatalf("JSON 序列化后无法解析：%v", err)
	}
	kb, ok := back.Shortcuts["toggle_window"]
	if !ok {
		t.Fatalf("round-trip 后应保留 toggle_window，原始 JSON: %s", raw)
	}
	if kb.ComboID != cfg.Shortcuts["toggle_window"].ComboID ||
		kb.KeyName != cfg.Shortcuts["toggle_window"].KeyName {
		t.Fatalf("round-trip 后 toggle_window 绑定不一致：%+v", kb)
	}
}
