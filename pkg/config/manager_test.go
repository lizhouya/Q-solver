package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
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

func TestUpdateAndPatchPersistMigratedCCRSettings(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		cm := isolatedManager(t)
		if err := cm.Load(); err != nil {
			t.Fatal(err)
		}
		if err := cm.UpdateFromJSON(`{"connectionMode":"ccr","apiKey":"test-only","baseURL":"http://127.0.0.1:3456/","apiProtocol":"anthropic_messages","model":"route-a"}`); err != nil {
			t.Fatal(err)
		}
		got := cm.Get()
		if got.ConnectionMode != ConnectionModeDirect || got.BaseURL != "http://127.0.0.1:3456/v1" || got.APIProtocol != "openai_chat" || got.EndpointPath != "/chat/completions" {
			t.Fatalf("update did not migrate CCR settings: %+v", got)
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
		if got.ConnectionMode != ConnectionModeDirect || got.BaseURL != "https://gateway.example.com/v1" || got.APIProtocol != "openai_chat" || got.EndpointPath != "/chat/completions" {
			t.Fatalf("patch did not migrate CCR settings: %+v", got)
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

func TestLoadLegacyProtocolWithoutEndpointUsesProtocolDefault(t *testing.T) {
	for _, tt := range []struct {
		name, protocol, want string
	}{
		{name: "responses", protocol: "openai_responses", want: "/responses"},
		{name: "anthropic", protocol: "anthropic_messages", want: "/v1/messages"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cm := isolatedManager(t)
			plain := []byte(`{"baseURL":"https://gateway.example.com","apiProtocol":"` + tt.protocol + `","model":"test-model"}`)
			data, err := encrypt(plain)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := cm.Load(); err != nil {
				t.Fatal(err)
			}
			if got := cm.Get().EndpointPath; got != tt.want {
				t.Fatalf("endpoint=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestUpdateProtocolWithoutEndpointUsesProtocolDefault(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.UpdateFromJSON(`{"baseURL":"https://gateway.example.com","apiProtocol":"anthropic_messages","model":"test-model"}`); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().EndpointPath; got != "/v1/messages" {
		t.Fatalf("endpoint=%q want=/v1/messages", got)
	}
}

func TestUpdatePreservesExplicitCrossProtocolEndpoint(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.UpdateFromJSON(`{"baseURL":"https://gateway.example.com","apiProtocol":"anthropic_messages","endpointPath":"/custom/chat/completions","model":"test-model"}`); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().EndpointPath; got != "/custom/chat/completions" {
		t.Fatalf("explicit endpoint changed: %q", got)
	}
}

func TestPatchValidatedDoesNotSaveOrNotifyWhenCandidateRejected(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if err := cm.Patch(func(cfg *Config) { cfg.Theme = "light" }); err != nil {
		t.Fatal(err)
	}
	before := cm.Get()
	beforeData, err := os.ReadFile(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeHash := sha256.Sum256(beforeData)
	notifications := 0
	cm.Subscribe(func(Config, Config) { notifications++ })

	err = cm.PatchValidated(
		func(cfg *Config) {
			cfg.Theme = "dark"
			cfg.APIKey = "candidate-only"
		},
		func(candidate Config) error {
			if candidate.Theme != "dark" {
				t.Fatalf("validator did not receive patched candidate: %+v", candidate)
			}
			return errors.New("test rejection")
		},
	)
	if err == nil {
		t.Fatal("rejected candidate must return error")
	}
	afterData, readErr := os.ReadFile(cm.configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if sha256.Sum256(afterData) != beforeHash {
		t.Fatal("rejected candidate changed persisted config")
	}
	if got := cm.Get(); !reflect.DeepEqual(got, before) {
		t.Fatalf("rejected candidate changed memory: before=%+v after=%+v", before, got)
	}
	if notifications != 0 {
		t.Fatalf("rejected candidate notified %d subscribers", notifications)
	}
}

func TestPatchValidatedSavesThenNotifiesOnce(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	notifications := 0
	cm.Subscribe(func(newConfig Config, oldConfig Config) {
		notifications++
		if oldConfig.Theme != "light" || newConfig.Theme != "dark" {
			t.Fatalf("unexpected notification values: old=%q new=%q", oldConfig.Theme, newConfig.Theme)
		}
		data, err := os.ReadFile(cm.configPath)
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
		if persisted.Theme != newConfig.Theme {
			t.Fatalf("subscriber ran before save: persisted=%q new=%q", persisted.Theme, newConfig.Theme)
		}
	})

	if err := cm.PatchValidated(
		func(cfg *Config) { cfg.Theme = "dark" },
		func(candidate Config) error {
			if candidate.Theme != "dark" {
				t.Fatalf("validator saw wrong candidate: %+v", candidate)
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if notifications != 1 || cm.Get().Theme != "dark" {
		t.Fatalf("notifications=%d theme=%q", notifications, cm.Get().Theme)
	}
}

func TestManagerRoundTripsCustomJSONFields(t *testing.T) {
	cm := isolatedManager(t)
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if err := cm.Patch(func(cfg *Config) {
		cfg.APIProtocol = "custom_json"
		cfg.BaseURL = "http://127.0.0.1:11434/generate"
		cfg.Model = "test-model"
		cfg.RequestTemplate = `{"model":"{{model}}","messages":"{{openai_messages}}"}`
		cfg.ResponseJSONPointer = "/answer"
		cfg.AuthHeader = "X-Test-Key"
		cfg.AuthPrefix = "Token "
		cfg.ExtraHeaders = map[string]string{"X-Route": "test-route"}
	}); err != nil {
		t.Fatal(err)
	}

	reloaded := &ConfigManager{config: NewDefaultConfig(), configPath: cm.configPath}
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	got := reloaded.Get()
	if got.APIProtocol != "custom_json" || got.BaseURL != "http://127.0.0.1:11434/generate" || got.Model != "test-model" {
		t.Fatalf("custom connection did not round trip: %+v", got)
	}
	if got.RequestTemplate == "" || got.ResponseJSONPointer != "/answer" || got.AuthHeader != "X-Test-Key" || got.AuthPrefix != "Token " {
		t.Fatalf("custom template fields did not round trip: %+v", got)
	}
	if got.ExtraHeaders["X-Route"] != "test-route" {
		t.Fatalf("custom headers did not round trip: %#v", got.ExtraHeaders)
	}
}

func TestLoadLegacyProtocolsDoesNotWriteOrMigrateBeforeSave(t *testing.T) {
	for _, protocolValue := range []string{"openai_responses", "anthropic_messages"} {
		t.Run(protocolValue, func(t *testing.T) {
			cm := isolatedManager(t)
			plain := []byte(`{"apiKey":"test-only","baseURL":"https://api.example.com","apiProtocol":"` + protocolValue + `","model":"test-model"}`)
			data, err := encrypt(plain)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256(data)
			if err := cm.Load(); err != nil {
				t.Fatal(err)
			}
			afterData, err := os.ReadFile(cm.configPath)
			if err != nil {
				t.Fatal(err)
			}
			if sha256.Sum256(afterData) != before {
				t.Fatal("loading legacy connection changed the encrypted file")
			}
			got := cm.Get()
			if got.APIProtocol != protocolValue || got.RequestTemplate != "" {
				t.Fatalf("legacy connection migrated during load: %+v", got)
			}
		})
	}
}

func profileUpdatePayload(t *testing.T, cfg Config, id, name string) string {
	t.Helper()
	payload, err := json.Marshal(struct {
		Config
		ConnectionProfileID   string `json:"connectionProfileId"`
		ConnectionProfileName string `json:"connectionProfileName"`
	}{Config: cfg, ConnectionProfileID: id, ConnectionProfileName: name})
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func TestUpdateFromJSONCreatesProfileFromLegacyConnection(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := NewDefaultConfig()
	cfg.BaseURL = "https://api.example.test/v1"
	cfg.APIProtocol = "openai_chat"
	cfg.EndpointPath = "/chat/completions"
	cfg.ResponseFormat = "text"
	cfg.Model = "model-a"
	if err := manager.UpdateFromJSON(profileUpdatePayload(t, cfg, LegacyConnectionProfileID, "")); err != nil {
		t.Fatal(err)
	}
	got := manager.Get()
	if len(got.ConnectionProfiles) != 1 || got.ActiveConnectionProfileID == "" || got.ActiveConnectionProfileID == LegacyConnectionProfileID {
		t.Fatalf("profile not created: %#v", got.ConnectionProfiles)
	}
	if got.ConnectionProfiles[0].ID != got.ActiveConnectionProfileID || got.BaseURL != got.ConnectionProfiles[0].BaseURL {
		t.Fatal("root mirror and active profile diverged")
	}
}

func TestUpdateFromJSONPreservesUnselectedProfilesAndUpdatesSelected(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	var first, second ConnectionProfile
	if err := manager.Patch(func(cfg *Config) {
		first, _ = UpsertConnectionProfile(cfg, ConnectionProfile{Name: "first", BaseURL: "https://first.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "first-model"})
		second, _ = UpsertConnectionProfile(cfg, ConnectionProfile{Name: "second", BaseURL: "https://second.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "second-model"})
		ApplyProfileToActiveConnection(cfg, first)
	}); err != nil {
		t.Fatal(err)
	}
	candidate := manager.Get()
	candidate.APIKey = "updated-test-key"
	candidate.BaseURL = "https://updated.test/v1"
	candidate.Model = "updated-model"
	if err := manager.UpdateFromJSON(profileUpdatePayload(t, candidate, second.ID, "second renamed")); err != nil {
		t.Fatal(err)
	}
	got := manager.Get()
	if len(got.ConnectionProfiles) != 2 || got.ActiveConnectionProfileID != second.ID {
		t.Fatalf("profiles=%#v active=%q", got.ConnectionProfiles, got.ActiveConnectionProfileID)
	}
	preserved, _ := FindConnectionProfile(got, first.ID)
	updated, _ := FindConnectionProfile(got, second.ID)
	if preserved.BaseURL != first.BaseURL || updated.Name != "second renamed" || updated.BaseURL != "https://updated.test/v1" || updated.APIKey != "updated-test-key" {
		t.Fatalf("preserved=%+v updated=%+v", preserved, updated)
	}
}

func TestUpdateFromJSONMissingProfileArrayCannotEraseStoredProfiles(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	var saved ConnectionProfile
	if err := manager.Patch(func(cfg *Config) {
		saved, _ = UpsertConnectionProfile(cfg, ConnectionProfile{Name: "saved", BaseURL: "https://saved.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "model-a"})
		ApplyProfileToActiveConnection(cfg, saved)
	}); err != nil {
		t.Fatal(err)
	}
	payload := `{"connectionProfileId":"` + saved.ID + `","connectionProfileName":"saved","baseURL":"https://saved.test/v1","apiProtocol":"openai_chat","endpointPath":"/chat/completions","responseFormat":"text","model":"model-b"}`
	if err := manager.UpdateFromJSON(payload); err != nil {
		t.Fatal(err)
	}
	got := manager.Get()
	if len(got.ConnectionProfiles) != 1 || got.ConnectionProfiles[0].Model != "model-b" {
		t.Fatalf("profiles=%#v", got.ConnectionProfiles)
	}
}

func TestUpdateFromJSONFailureKeepsProfilesRootAndSubscribersUnchanged(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	var saved ConnectionProfile
	if err := manager.Patch(func(cfg *Config) {
		saved, _ = UpsertConnectionProfile(cfg, ConnectionProfile{Name: "saved", BaseURL: "https://saved.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "model-a"})
		ApplyProfileToActiveConnection(cfg, saved)
	}); err != nil {
		t.Fatal(err)
	}
	before := manager.Get()
	calls := 0
	manager.Subscribe(func(Config, Config) { calls++ })
	manager.configPath = t.TempDir()
	candidate := before
	candidate.Model = "must-not-save"
	if err := manager.UpdateFromJSON(profileUpdatePayload(t, candidate, saved.ID, saved.Name)); err == nil {
		t.Fatal("save to directory must fail")
	}
	if calls != 0 || !reflect.DeepEqual(manager.Get(), before) {
		t.Fatalf("failed save changed state: calls=%d", calls)
	}
}

func TestLoadLegacyConnectionDoesNotWriteProfileOrRewriteEncryptedFile(t *testing.T) {
	manager := isolatedManager(t)
	plain := []byte(`{"baseURL":"https://legacy.test/v1","apiProtocol":"openai_chat","model":"legacy-model"}`)
	data, err := encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manager.configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(data)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	got := manager.Get()
	if len(got.ConnectionProfiles) != 0 || got.ActiveConnectionProfileID != "" {
		t.Fatalf("legacy load persisted profiles: %#v", got.ConnectionProfiles)
	}
	after, err := os.ReadFile(manager.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(after) != before {
		t.Fatal("legacy load rewrote encrypted file")
	}
}

func TestGetAndCloneConfigSnapshotDeepCopyConnectionProfiles(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Patch(func(cfg *Config) {
		saved, _ := UpsertConnectionProfile(cfg, ConnectionProfile{Name: "saved", BaseURL: "https://saved.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ExtraHeaders: map[string]string{"X-Test": "original"}, ResponseFormat: "text"})
		ApplyProfileToActiveConnection(cfg, saved)
	}); err != nil {
		t.Fatal(err)
	}
	got := manager.Get()
	got.ConnectionProfiles[0].Name = "mutated"
	got.ConnectionProfiles[0].ExtraHeaders["X-Test"] = "mutated"
	unchanged := manager.Get()
	if unchanged.ConnectionProfiles[0].Name == "mutated" || unchanged.ConnectionProfiles[0].ExtraHeaders["X-Test"] == "mutated" {
		t.Fatal("manager Get leaked profile aliases")
	}
}

func TestConnectionProfilesPersistEncryptedWithoutPlaintextSentinels(t *testing.T) {
	manager := isolatedManager(t)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := NewDefaultConfig()
	cfg.APIKey = "profile-key-sentinel"
	cfg.BaseURL = "https://secure.test/generate"
	cfg.APIProtocol = "custom_json"
	cfg.RequestTemplate = `{"messages":"{{openai_messages}}","marker":"profile-template-sentinel"}`
	cfg.ResponseJSONPointer = "/answer"
	cfg.AuthHeader = "X-Test-Key"
	cfg.AuthPrefix = "profile-prefix-sentinel "
	cfg.ExtraHeaders = map[string]string{"X-Test": "profile-header-sentinel"}
	cfg.Model = "secure-model"
	if err := manager.UpdateFromJSON(profileUpdatePayload(t, cfg, "", "secure")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(manager.configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range [][]byte{[]byte("profile-key-sentinel"), []byte("profile-header-sentinel"), []byte("profile-template-sentinel"), []byte("profile-prefix-sentinel")} {
		if bytes.Contains(raw, sentinel) {
			t.Fatal("encrypted config contains plaintext connection sentinel")
		}
	}
	plain, err := decrypt(raw)
	if err != nil {
		t.Fatal(err)
	}
	var persisted Config
	if err := json.Unmarshal(plain, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.ConnectionProfiles) != 1 || persisted.ConnectionProfiles[0].APIKey != cfg.APIKey {
		t.Fatal("encrypted profile did not round trip")
	}
}
