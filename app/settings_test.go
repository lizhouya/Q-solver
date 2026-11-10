package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"Q-Solver/pkg/common"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
)

func TestGetSettingsLoadsBeforeStartup(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", root)
	seed := config.NewConfigManager()
	if err := seed.UpdateFromJSON(`{"apiKey":"test-only","baseURL":"http://localhost/v1","model":"test-model"}`); err != nil {
		t.Fatal(err)
	}
	a := &App{configManager: config.NewConfigManager()}
	got, err := a.GetSettings()
	if err != nil {
		t.Fatal("GetSettings before Startup must load saved config")
	}
	if got.ActiveConnectionProfileID == "" {
		t.Fatal("active profile ID was not loaded")
	}
	if got.APIKey != "" || got.BaseURL != "" || got.Model != "" || got.ExtraHeaders != nil || got.ConnectionProfiles != nil {
		t.Fatalf("GetSettings exposed connection fields: %+v", got)
	}
}

func TestGetSettingsReportsReadFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", root)
	a := &App{configManager: config.NewConfigManager()}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, common.AppName, "config"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetSettings(); err == nil {
		t.Fatal("read error must reach Wails, not default empty settings")
	}
}

func newConnectionSettingsTestApp(t *testing.T) *App {
	t.Helper()
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", root)
	manager := config.NewConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Patch(func(cfg *config.Config) {
		cfg.APIKey = "saved-key"
		cfg.BaseURL = "https://saved.example.com/v1"
		cfg.Model = "saved-model"
		cfg.APIProtocol = "openai_chat"
		cfg.Prompt = "keep-prompt"
		cfg.DomainId = "keep-domain"
		cfg.Opacity = 0.73
		cfg.KeepContext = true
		cfg.OutputMode = "dual"
		cfg.ResumePath = "keep-resume"
		cfg.Theme = "dark"
	}); err != nil {
		t.Fatal(err)
	}
	return &App{configManager: manager, llmService: llm.NewService(manager.Get(), manager)}
}

func TestSaveConnectionSettingsPatchesOnlyConnectionFields(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	before := a.configManager.Get()
	draft := ConnectionDraft{
		ConnectionMode: "direct",
		APIKey:         "new-key",
		BaseURL:        "https://new.example.com/api",
		Model:          "new-model",
		APIProtocol:    "openai_responses",
		EndpointPath:   "/custom/responses",
		ExtraHeaders:   map[string]string{"X-Route": "route-a"},
		ResponseFormat: "text",
	}
	result := a.SaveConnectionSettings(ConnectionProfileDraft{ID: config.LegacyConnectionProfileID, Connection: draft})
	if !result.Success || result.Error != nil || result.Connection == nil {
		t.Fatalf("save failed: %+v", result)
	}
	after := a.configManager.Get()
	if after.APIKey != draft.APIKey || after.BaseURL != draft.BaseURL || after.Model != draft.Model || after.APIProtocol != draft.APIProtocol || after.EndpointPath != draft.EndpointPath || after.ExtraHeaders["X-Route"] != "route-a" {
		t.Fatalf("connection fields not saved: %+v", after)
	}
	if after.Prompt != before.Prompt || after.DomainId != before.DomainId || after.Opacity != before.Opacity || after.KeepContext != before.KeepContext || after.OutputMode != before.OutputMode || after.ResumePath != before.ResumePath || after.Theme != before.Theme || !reflect.DeepEqual(after.Shortcuts, before.Shortcuts) {
		t.Fatalf("non-connection fields changed: before=%+v after=%+v", before, after)
	}
}

func TestSaveConnectionSettingsFailureKeepsSavedConfigAndProvider(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	beforeConfig := a.configManager.Get()
	beforeProvider := a.llmService.GetProvider()
	result := a.SaveConnectionSettings(ConnectionProfileDraft{ID: config.LegacyConnectionProfileID, Connection: ConnectionDraft{
		ConnectionMode: "direct", BaseURL: "https://new.example.com/v1", Model: "new-model",
		APIProtocol: "openai_chat", EndpointPath: "/chat?unsafe=1",
	}})
	if result.Success || result.Error == nil || result.Error.Code != config.ProfileErrorInvalid {
		t.Fatalf("invalid save result: %+v", result)
	}
	if !reflect.DeepEqual(a.configManager.Get(), beforeConfig) || a.llmService.GetProvider() != beforeProvider {
		t.Fatal("failed save changed config or active provider")
	}
}

func TestSaveConnectionResultDoesNotExposeSecretValues(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	const keySentinel = "test-private-api-key"
	const headerSentinel = "test-private-header-value"
	const templateSentinel = "test-private-template-marker"
	const authPrefixSentinel = "test-private-auth-prefix "
	result := a.SaveConnectionSettings(ConnectionProfileDraft{ID: config.LegacyConnectionProfileID, Name: "secure", Connection: ConnectionDraft{
		ConnectionMode: "direct", APIKey: keySentinel, BaseURL: "https://new.example.com/generate",
		Model: "new-model", APIProtocol: "custom_json",
		ExtraHeaders: map[string]string{"X-Private-Route": headerSentinel}, ResponseFormat: "text",
		RequestTemplate:     `{"marker":"` + templateSentinel + `","messages":"{{openai_messages}}"}`,
		ResponseJSONPointer: "/answer", AuthHeader: "X-Test-Key", AuthPrefix: authPrefixSentinel,
	}})
	if !result.Success || result.Connection == nil || !result.Connection.APIKeySet || !result.Connection.ExtraHeadersSet || !result.Connection.TemplateSet {
		t.Fatalf("unexpected safe result: %+v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{keySentinel, headerSentinel, templateSentinel, authPrefixSentinel, "/answer", "X-Test-Key"} {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatalf("save result leaked private connection value: %s", sentinel)
		}
	}
	got := a.configManager.Get()
	if got.APIProtocol != "custom_json" || got.RequestTemplate == "" || got.ResponseJSONPointer != "/answer" || got.AuthHeader != "X-Test-Key" || got.AuthPrefix != authPrefixSentinel {
		t.Fatalf("custom template fields were not persisted: %+v", got)
	}
}

func TestSaveConnectionSettingsRejectsInvalidTemplateAtomically(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	beforeConfig := a.configManager.Get()
	beforeProvider := a.llmService.GetProvider()
	result := a.SaveConnectionSettings(ConnectionProfileDraft{ID: config.LegacyConnectionProfileID, Connection: ConnectionDraft{
		ConnectionMode: "direct", BaseURL: "https://new.example.com/generate", Model: "new-model",
		APIProtocol: "custom_json", RequestTemplate: `{"messages":"prefix-{{openai_messages}}"}`,
		ResponseJSONPointer: "/answer", AuthHeader: "Authorization", AuthPrefix: "Bearer ",
	}})
	if result.Success || result.Error == nil || result.Error.Code != config.ProfileErrorInvalid {
		t.Fatalf("invalid template save result: %+v", result)
	}
	if !reflect.DeepEqual(a.configManager.Get(), beforeConfig) || a.llmService.GetProvider() != beforeProvider {
		t.Fatal("invalid template save changed config or active provider")
	}
}

func TestUpdateSettingsReturnsSafeErrorWithoutSecret(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	result := a.UpdateSettings(`{"apiKey":"update-private-key","apiProtocol":"invalid-protocol"}`)
	if result != "设置保存失败，原配置未修改" || strings.Contains(result, "update-private-key") || strings.Contains(result, "invalid-protocol") {
		t.Fatalf("unsafe result=%q", result)
	}
}
