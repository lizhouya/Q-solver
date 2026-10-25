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

func TestDefaultClearMobileDisplayShortcut(t *testing.T) {
	cfg := NewDefaultConfig()
	kb, ok := cfg.Shortcuts["clear_mobile_display"]
	if runtime.GOOS == "darwin" {
		if ok {
			t.Fatal("macOS 默认快捷键不得包含 clear_mobile_display")
		}
		return
	}
	if runtime.GOOS != "windows" {
		t.Skipf("未覆盖平台 %s", runtime.GOOS)
	}
	if !ok || kb.ComboID != "67+160" || kb.KeyName != "Shift+C" {
		t.Fatalf("Windows 清屏默认快捷键应为 Shift+C(67+160)，实际 %+v", kb)
	}
}

func TestDefaultStartNextExamShortcut(t *testing.T) {
	windowsShortcuts := getDefaultShortcutsForOS("windows")
	kb, ok := windowsShortcuts["start_next_exam"]
	if !ok || kb.ComboID != "78+160+162" || kb.KeyName != "Ctrl+Shift+N" {
		t.Fatalf("Windows 开始下一套默认快捷键应为 Ctrl+Shift+N(78+160+162)，实际 %+v", kb)
	}

	if _, ok := getDefaultShortcutsForOS("darwin")["start_next_exam"]; ok {
		t.Fatal("macOS 默认快捷键不得包含 start_next_exam")
	}
}

func TestDefaultWindowsShortcutComboIDsAreUnique(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("Windows-only test on %s", runtime.GOOS)
	}
	seen := make(map[string]string)
	for action, binding := range NewDefaultConfig().Shortcuts {
		if binding.ComboID == "" {
			continue
		}
		if other, exists := seen[binding.ComboID]; exists {
			t.Fatalf("默认快捷键冲突：%s 与 %s 共用 %s", other, action, binding.ComboID)
		}
		seen[binding.ComboID] = action
	}
	if owner := seen["78+160+162"]; owner != "start_next_exam" {
		t.Fatalf("Ctrl+Shift+N 默认组合应唯一归 start_next_exam，实际归属 %q", owner)
	}
}
