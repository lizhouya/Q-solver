package app

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
)

func seedTwoConnectionProfiles(t *testing.T, a *App) []string {
	t.Helper()
	var ids []string
	var seedErr error
	if err := a.configManager.Patch(func(cfg *config.Config) {
		first, err := config.UpsertConnectionProfile(cfg, config.ConnectionProfile{
			Name: "A 连接", APIKey: "test-private-key", BaseURL: "https://a.example.test/v1",
			APIProtocol: "openai_chat", EndpointPath: "/chat/completions",
			ExtraHeaders:   map[string]string{"X-Test": "test-private-header"},
			ResponseFormat: "text", Model: "model-a",
		})
		if err != nil {
			seedErr = err
			return
		}
		second, err := config.UpsertConnectionProfile(cfg, config.ConnectionProfile{
			Name: "B 连接", BaseURL: "https://b.example.test/generate", APIProtocol: "custom_json",
			RequestTemplate:     `{"messages":"{{openai_messages}}","marker":"test-private-template"}`,
			ResponseJSONPointer: "/answer", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Model: "model-b",
		})
		if err != nil {
			seedErr = err
			return
		}
		config.ApplyProfileToActiveConnection(cfg, first)
		ids = []string{first.ID, second.ID}
	}); err != nil {
		t.Fatal(err)
	}
	if seedErr != nil {
		t.Fatal(seedErr)
	}
	return ids
}

func TestListConnectionProfilesReturnsSafeSummariesOnly(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	seedTwoConnectionProfiles(t, a)
	result := a.ListConnectionProfiles()
	if result.Error != nil || len(result.Profiles) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if !result.Profiles[0].Active || result.Profiles[0].AddressLabel != "a.example.test" || !result.Profiles[0].APIKeySet {
		t.Fatalf("summary=%+v", result.Profiles[0])
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{"test-private-key", "test-private-header", "test-private-template", "https://"} {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatal("summary leaked private connection data")
		}
	}
}

func TestGetConnectionProfileReturnsOnlyRequestedDraftWithoutChangingRuntime(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	ids := seedTwoConnectionProfiles(t, a)
	beforeConfig := a.configManager.Get()
	beforeProvider := a.llmService.GetProvider()
	result := a.GetConnectionProfile(ids[1])
	if !result.Success || result.Profile == nil || result.Profile.ID != ids[1] || result.Profile.Connection.BaseURL != "https://b.example.test/generate" {
		t.Fatalf("result=%+v", result)
	}
	if !reflect.DeepEqual(a.configManager.Get(), beforeConfig) || a.llmService.GetProvider() != beforeProvider {
		t.Fatal("targeted read changed active runtime")
	}
	result.Profile.Connection.ExtraHeaders["X-Mutated"] = "yes"
	stored, _ := config.FindConnectionProfile(a.configManager.Get(), ids[1])
	if _, ok := stored.ExtraHeaders["X-Mutated"]; ok {
		t.Fatal("draft aliases stored headers")
	}
}

func TestLegacyConnectionAppearsAsTemporaryActiveSummaryAndTargetedDraft(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	list := a.ListConnectionProfiles()
	if list.Error != nil || len(list.Profiles) != 1 || list.Profiles[0].ID != config.LegacyConnectionProfileID || !list.Profiles[0].Active {
		t.Fatalf("list=%+v", list)
	}
	result := a.GetConnectionProfile(config.LegacyConnectionProfileID)
	if !result.Success || result.Profile == nil || result.Profile.Name != "" || result.Profile.Connection.APIKey != "saved-key" {
		t.Fatalf("result=%+v", result)
	}
	if len(a.configManager.Get().ConnectionProfiles) != 0 {
		t.Fatal("legacy read persisted a profile")
	}
}

func TestSaveConnectionSettingsCreatesUpdatesAndActivatesProfileAtomically(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	result := a.SaveConnectionSettings(ConnectionProfileDraft{
		Name: "",
		Connection: ConnectionDraft{
			ConnectionMode: "direct", APIKey: "new-key", BaseURL: "https://new.example.test/v1", Model: "new-model",
			APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ExtraHeaders: map[string]string{}, ResponseFormat: "text",
		},
	})
	if !result.Success || result.Profile == nil || result.Connection == nil || result.Profile.ID == "" || !result.Profile.Active {
		t.Fatalf("result=%+v", result)
	}
	got := a.configManager.Get()
	if len(got.ConnectionProfiles) != 1 || got.ActiveConnectionProfileID != result.Profile.ID || got.APIKey != "new-key" {
		t.Fatalf("config=%+v", got)
	}
	update := ConnectionProfileDraft{ID: result.Profile.ID, Name: "renamed", Connection: ConnectionDraft{
		ConnectionMode: "direct", APIKey: "rotated-key", BaseURL: "https://new.example.test/v1", Model: "new-model-2",
		APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ExtraHeaders: map[string]string{}, ResponseFormat: "text",
	}}
	updated := a.SaveConnectionSettings(update)
	if !updated.Success || updated.Profile == nil || updated.Profile.ID != result.Profile.ID || updated.Profile.Name != "renamed" {
		t.Fatalf("updated=%+v", updated)
	}
	if len(a.configManager.Get().ConnectionProfiles) != 1 {
		t.Fatal("update appended a duplicate profile")
	}
}

func TestSaveConnectionSettingsFailureLeavesProfilesAndProviderUnchanged(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	ids := seedTwoConnectionProfiles(t, a)
	before := a.configManager.Get()
	beforeProvider := a.llmService.GetProvider()
	result := a.SaveConnectionSettings(ConnectionProfileDraft{
		ID: ids[1], Name: "B 连接",
		Connection: ConnectionDraft{ConnectionMode: "direct", BaseURL: "https://bad.example.test/v1", Model: "bad", APIProtocol: "openai_chat", EndpointPath: "/chat?unsafe=1"},
	})
	if result.Success || result.Error == nil {
		t.Fatalf("result=%+v", result)
	}
	if !reflect.DeepEqual(a.configManager.Get(), before) || a.llmService.GetProvider() != beforeProvider {
		t.Fatal("failed save changed profiles or provider")
	}
}

func TestSaveConnectionResultContainsGeneratedIdentityButNoSecret(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	result := a.SaveConnectionSettings(ConnectionProfileDraft{
		Name: "secure",
		Connection: ConnectionDraft{
			ConnectionMode: "direct", APIKey: "save-private-key", BaseURL: "https://secure.example.test/generate", Model: "secure-model",
			APIProtocol: "custom_json", ExtraHeaders: map[string]string{"X-Test": "save-private-header"}, ResponseFormat: "text",
			RequestTemplate: `{"messages":"{{openai_messages}}","marker":"save-private-template"}`, ResponseJSONPointer: "/answer",
			AuthHeader: "X-Test-Key", AuthPrefix: "save-private-prefix ",
		},
	})
	if !result.Success || result.Profile == nil || !result.Profile.APIKeySet || !result.Profile.TemplateSet {
		t.Fatalf("result=%+v", result)
	}
	encoded, _ := json.Marshal(result)
	for _, sentinel := range []string{"save-private-key", "save-private-header", "save-private-template", "save-private-prefix", "/answer", "X-Test-Key"} {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatal("save result leaked private connection data")
		}
	}
}

func TestDeleteConnectionProfileAllowsOnlyInactiveAndKeepsRootConnection(t *testing.T) {
	a := newConnectionSettingsTestApp(t)
	ids := seedTwoConnectionProfiles(t, a)
	before := a.configManager.Get()
	activeResult := a.DeleteConnectionProfile(ids[0])
	if activeResult.Success || activeResult.Error == nil || activeResult.Error.Code != config.ProfileErrorActiveDeleteForbidden {
		t.Fatalf("active result=%+v", activeResult)
	}
	result := a.DeleteConnectionProfile(ids[1])
	if !result.Success || result.DeletedID != ids[1] {
		t.Fatalf("delete result=%+v", result)
	}
	after := a.configManager.Get()
	if len(after.ConnectionProfiles) != 1 || after.ActiveConnectionProfileID != ids[0] || after.BaseURL != before.BaseURL || after.APIKey != before.APIKey {
		t.Fatalf("after=%+v", after)
	}
}

func TestConnectionAddressLabelDropsSchemePathAndQuery(t *testing.T) {
	for input, want := range map[string]string{
		"https://api.example.test/v1/path?secret=value": "api.example.test",
		"http://127.0.0.1:3456/v1":                      "127.0.0.1:3456",
		"https://[::1]:8443/v1":                         "[::1]:8443",
	} {
		if got := connectionAddressLabel(input); got != want {
			t.Fatalf("label(%q)=%q want %q", input, got, want)
		}
	}
}
