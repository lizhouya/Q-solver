package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"Q-Solver/pkg/shortcut"
)

func isolatedManager(t *testing.T) *ConfigManager {
	t.Helper()
	return &ConfigManager{config: NewDefaultConfig(), configPath: filepath.Join(t.TempDir(), "config")}
}

func writeTestConfig(t *testing.T, cm *ConfigManager) {
	t.Helper()
	data, err := encrypt([]byte(`{"apiKey":"test-only","baseURL":"http://localhost/v1","model":"test-model"}`))
	if err != nil {
		t.Fatal("fixture encryption failed")
	}
	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		t.Fatal("fixture write failed")
	}
}

func TestLoadPreservesCredentialsAndIsIdempotent(t *testing.T) {
	cm := isolatedManager(t)
	writeTestConfig(t, cm)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if cm.Get().APIKey != "test-only" || cm.Get().Model != "test-model" {
		t.Fatal("settings not loaded")
	}
	if err := os.WriteFile(cm.configPath, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cm.Load(); err != nil {
		t.Fatal("successful load should be idempotent")
	}
	if cm.Get().APIKey != "test-only" {
		t.Fatal("loaded credentials were reset")
	}
}

func TestFailedLoadCannotOverwriteFile(t *testing.T) {
	cm := isolatedManager(t)
	original := []byte("broken encrypted config")
	if err := os.WriteFile(cm.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cm.Load(); err == nil {
		t.Fatal("corrupt config must return an error")
	}
	if err := cm.Save(); err == nil {
		t.Fatal("save before successful load must fail")
	}
	if err := cm.UpdateFromJSON(`{"apiKey":"replacement"}`); err == nil {
		t.Fatal("update must not overwrite unread config")
	}
	if err := cm.Patch(func(c *Config) { c.Theme = "dark" }); err == nil {
		t.Fatal("patch must not overwrite unread config")
	}
	got, err := os.ReadFile(cm.configPath)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("original file modified")
	}
	writeTestConfig(t, cm)
	if err := cm.Load(); err != nil {
		t.Fatal("fixed file must allow retry")
	}
}

func TestMissingConfigAllowsFirstSave(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if err := cm.UpdateFromJSON(`{"apiKey":"test-only"}`); err != nil {
		t.Fatal(err)
	}
	if err := cm.Save(); err != nil {
		t.Fatal(err)
	}
}

func writeShortcutConfig(t *testing.T, cm *ConfigManager, shortcuts map[string]shortcut.KeyBinding) {
	t.Helper()
	plain, err := json.Marshal(Config{Shortcuts: shortcuts})
	if err != nil {
		t.Fatal(err)
	}
	data, err := encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMigratesStartNextExamShortcutSafely(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("Windows-only migration test on %s", runtime.GOOS)
	}

	defaultBinding := shortcut.KeyBinding{ComboID: "78+160+162", KeyName: "Ctrl+Shift+N"}

	t.Run("missing action with free default", func(t *testing.T) {
		cm := isolatedManager(t)
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"screenshot": {ComboID: "119", KeyName: "F8"},
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		if got := cm.Get().Shortcuts["start_next_exam"]; got != defaultBinding {
			t.Fatalf("action 缺失且默认组合空闲时应补默认：want %+v, got %+v", defaultBinding, got)
		}
	})

	t.Run("missing action with occupied default", func(t *testing.T) {
		cm := isolatedManager(t)
		occupied := shortcut.KeyBinding{ComboID: defaultBinding.ComboID, KeyName: "Legacy Ctrl+Shift+N"}
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"legacy_action": occupied,
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		got := cm.Get().Shortcuts
		if binding := got["legacy_action"]; binding != occupied {
			t.Fatalf("迁移不得抢占其他 action 的默认组合：want %+v, got %+v", occupied, binding)
		}
		if binding := got["start_next_exam"]; binding != (shortcut.KeyBinding{}) {
			t.Fatalf("默认组合被占用时 start_next_exam 应保持未绑定，实际 %+v", binding)
		}
	})

	for _, test := range []struct {
		name    string
		binding shortcut.KeyBinding
	}{
		{name: "explicit empty", binding: shortcut.KeyBinding{}},
		{name: "explicit custom", binding: shortcut.KeyBinding{ComboID: "77+160+162", KeyName: "Ctrl+Shift+M"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cm := isolatedManager(t)
			writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
				"start_next_exam": test.binding,
			})
			if err := cm.Load(); err != nil {
				t.Fatal(err)
			}
			if got := cm.Get().Shortcuts["start_next_exam"]; got != test.binding {
				t.Fatalf("显式绑定不得被迁移覆盖：want %+v, got %+v", test.binding, got)
			}
		})
	}
}

func TestLoadMigratesClearMobileDisplayShortcut(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("Windows-only migration test on %s", runtime.GOOS)
	}

	t.Run("free default", func(t *testing.T) {
		cm := isolatedManager(t)
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"screenshot": {ComboID: "119", KeyName: "F8"},
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		got := cm.Get().Shortcuts["clear_mobile_display"]
		if got.ComboID != "67+160" || got.KeyName != "Shift+C" {
			t.Fatalf("空闲默认组合应自动迁移，实际 %+v", got)
		}
	})

	t.Run("occupied default", func(t *testing.T) {
		cm := isolatedManager(t)
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"legacy_action": {ComboID: "67+160", KeyName: "Legacy Shift+C"},
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		got := cm.Get().Shortcuts
		if binding := got["legacy_action"]; binding != (shortcut.KeyBinding{}) {
			t.Fatalf("默认 Shift+C 必须从旧 action 迁移给清屏，旧绑定应清空，实际 %+v", binding)
		}
		if binding := got["clear_mobile_display"]; binding.ComboID != "67+160" || binding.KeyName != "Shift+C" {
			t.Fatalf("清屏应取得默认 Shift+C，实际 %+v", binding)
		}
	})

	t.Run("persisted empty repairs legacy collision", func(t *testing.T) {
		cm := isolatedManager(t)
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"clear_mobile_display": {},
			"toggle_window":        {ComboID: "67+160", KeyName: "Shift+C"},
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		got := cm.Get().Shortcuts
		if binding := got["toggle_window"]; binding != (shortcut.KeyBinding{}) {
			t.Fatalf("旧版持久化空清屏绑定时必须清除隐藏窗口冲突，实际 %+v", binding)
		}
		if binding := got["clear_mobile_display"]; binding != (shortcut.KeyBinding{ComboID: "67+160", KeyName: "Shift+C"}) {
			t.Fatalf("旧版空绑定应修复为清屏默认值，实际 %+v", binding)
		}
	})

	t.Run("explicit default removes duplicate", func(t *testing.T) {
		cm := isolatedManager(t)
		writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
			"clear_mobile_display": {ComboID: "67+160", KeyName: "Shift+C"},
			"toggle_window":        {ComboID: "67+160", KeyName: "Shift+C"},
		})
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		got := cm.Get().Shortcuts
		if binding := got["toggle_window"]; binding != (shortcut.KeyBinding{}) {
			t.Fatalf("显式默认清屏绑定应清除重复的隐藏窗口绑定，实际 %+v", binding)
		}
		if binding := got["clear_mobile_display"]; binding.ComboID != "67+160" {
			t.Fatalf("清屏默认绑定应保留，实际 %+v", binding)
		}
	})

	for _, test := range []struct {
		name    string
		binding shortcut.KeyBinding
	}{
		{name: "explicit custom", binding: shortcut.KeyBinding{ComboID: "67+162", KeyName: "Ctrl+C"}},
		{name: "explicit empty", binding: shortcut.KeyBinding{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cm := isolatedManager(t)
			writeShortcutConfig(t, cm, map[string]shortcut.KeyBinding{
				"clear_mobile_display": test.binding,
			})
			if err := cm.Load(); err != nil {
				t.Fatal(err)
			}
			if got := cm.Get().Shortcuts["clear_mobile_display"]; got != test.binding {
				t.Fatalf("显式绑定不得被迁移覆盖：want %+v, got %+v", test.binding, got)
			}
		})
	}
}

func TestLoadLegacyConfigUsesOpenAIChatProtocol(t *testing.T) {
	cm := isolatedManager(t)
	writeTestConfig(t, cm)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().APIProtocol; got != "openai_chat" {
		t.Fatalf("旧配置应解释为 openai_chat，实际 %q", got)
	}
}

func TestConfigRejectsUnknownAPIProtocol(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.APIProtocol = "claude-opus-5"
	if err := cfg.Validate(); err == nil {
		t.Fatal("未知非空协议必须被拒绝")
	}

	cfg.APIProtocol = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("空协议必须兼容旧配置: %v", err)
	}
}

func TestUpdateFailureDoesNotPublishCandidate(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	before := cm.Get()
	beforeOld := cm.oldConfig
	notifications := 0
	cm.Subscribe(func(Config, Config) { notifications++ })

	cm.configPath = t.TempDir() // 目录不能被当作配置文件覆盖，强制持久化失败。
	err := cm.UpdateFromJSON(`{"apiKey":"replacement","apiProtocol":"openai_responses"}`)
	if err == nil {
		t.Fatal("持久化失败时 UpdateFromJSON 必须返回错误")
	}
	if got := cm.Get(); !reflect.DeepEqual(got, before) {
		t.Fatalf("持久化失败后内存配置发生变化: before=%+v after=%+v", before, got)
	}
	if !reflect.DeepEqual(cm.oldConfig, beforeOld) {
		t.Fatal("持久化失败后 oldConfig 不得变化")
	}
	if notifications != 0 {
		t.Fatalf("持久化失败后不得通知 subscriber，实际 %d 次", notifications)
	}
}

func TestPatchFailureDoesNotPublishCandidate(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	before := cm.Get()
	notifications := 0
	cm.Subscribe(func(Config, Config) { notifications++ })
	cm.configPath = t.TempDir()

	err := cm.Patch(func(cfg *Config) {
		cfg.Theme = "dark"
	})
	if err == nil {
		t.Fatal("持久化失败时 Patch 必须返回错误")
	}
	if got := cm.Get(); !reflect.DeepEqual(got, before) {
		t.Fatalf("持久化失败后内存配置发生变化: before=%+v after=%+v", before, got)
	}
	if notifications != 0 {
		t.Fatalf("持久化失败后不得通知 subscriber，实际 %d 次", notifications)
	}
}

func TestSuccessfulUpdatePersistsBeforeSubscriber(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}

	var persisted Config
	cm.Subscribe(func(newConfig Config, oldConfig Config) {
		data, err := os.ReadFile(cm.configPath)
		if err != nil {
			t.Fatalf("subscriber 执行前配置应已持久化: %v", err)
		}
		plain, err := decrypt(data)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(plain, &persisted); err != nil {
			t.Fatal(err)
		}
		if persisted.APIProtocol != newConfig.APIProtocol || oldConfig.APIProtocol != "openai_chat" {
			t.Fatalf("subscriber 收到的配置顺序错误: old=%q new=%q persisted=%q", oldConfig.APIProtocol, newConfig.APIProtocol, persisted.APIProtocol)
		}
	})

	if err := cm.UpdateFromJSON(`{"apiKey":"test-only","apiProtocol":"openai_responses"}`); err != nil {
		t.Fatal(err)
	}
	if persisted.APIProtocol != "openai_responses" {
		t.Fatalf("subscriber 看到的磁盘配置错误: %q", persisted.APIProtocol)
	}
}

func TestLoadLegacyConfigNormalizesConnectionInMemoryWithoutWriting(t *testing.T) {
	cm := isolatedManager(t)
	writeTestConfig(t, cm)
	before, err := os.ReadFile(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	loaded := cm.Get()
	if loaded.ConnectionMode != ConnectionModeDirect || loaded.APIProtocol != "openai_chat" {
		t.Fatalf("legacy config not normalized in memory: %+v", loaded)
	}
	after, err := os.ReadFile(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	afterInfo, err := os.Stat(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatal("Load must not rewrite a legacy config")
	}
}

func TestUpdateAndPatchPersistCanonicalCCRSettings(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		cm := isolatedManager(t)
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		if err := cm.UpdateFromJSON(`{"connectionMode":"ccr","apiKey":"test-only","baseURL":"http://127.0.0.1:3456/","apiProtocol":"anthropic_messages","model":"route-a"}`); err != nil {
			t.Fatal(err)
		}
		got := cm.Get()
		if got.ConnectionMode != ConnectionModeCCR || got.BaseURL != "http://127.0.0.1:3456/v1" || got.APIProtocol != "openai_chat" {
			t.Fatalf("update did not publish canonical CCR settings: %+v", got)
		}
		assertPersistedConnectionSettings(t, cm.configPath, got)
	})

	t.Run("patch", func(t *testing.T) {
		cm := isolatedManager(t)
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		if err := cm.Patch(func(cfg *Config) {
			cfg.ConnectionMode = ConnectionModeCCR
			cfg.BaseURL = "https://gateway.example.com"
			cfg.APIProtocol = "openai_responses"
		}); err != nil {
			t.Fatal(err)
		}
		got := cm.Get()
		if got.ConnectionMode != ConnectionModeCCR || got.BaseURL != "https://gateway.example.com/v1" || got.APIProtocol != "openai_chat" {
			t.Fatalf("patch did not publish canonical CCR settings: %+v", got)
		}
		assertPersistedConnectionSettings(t, cm.configPath, got)
	})
}

func assertPersistedConnectionSettings(t *testing.T, path string, want Config) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := decrypt(data)
	if err != nil {
		t.Fatal(err)
	}
	var persisted Config
	if err := json.Unmarshal(plain, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ConnectionMode != want.ConnectionMode || persisted.BaseURL != want.BaseURL || persisted.APIProtocol != want.APIProtocol {
		t.Fatalf("persisted connection settings are not canonical: %+v", persisted)
	}
}

func TestCCRNormalizationFailureDoesNotPublishCandidate(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	before := cm.Get()
	notifications := 0
	cm.Subscribe(func(Config, Config) { notifications++ })

	err := cm.Patch(func(cfg *Config) {
		cfg.ConnectionMode = ConnectionModeCCR
		cfg.BaseURL = "https://gateway.example.com/v1/models"
	})
	if err == nil {
		t.Fatal("invalid CCR endpoint must be rejected")
	}
	if got := cm.Get(); !reflect.DeepEqual(got, before) {
		t.Fatalf("normalization failure changed config: before=%+v after=%+v", before, got)
	}
	if notifications != 0 {
		t.Fatalf("normalization failure notified subscribers %d times", notifications)
	}
	if _, err := os.Stat(cm.configPath); !os.IsNotExist(err) {
		t.Fatalf("normalization failure should not write a config file: %v", err)
	}
}
