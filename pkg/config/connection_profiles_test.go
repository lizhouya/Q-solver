package config

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNormalizeConnectionProfilesLeavesLegacyRootConnectionUnpersisted(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.APIKey = "test-only-key"
	cfg.BaseURL = "https://legacy.example.test/v1"
	cfg.Model = "legacy-model"

	got, err := NormalizeConnectionProfiles(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveConnectionProfileID != "" || len(got.ConnectionProfiles) != 0 {
		t.Fatalf("legacy load must remain in-memory only: %#v", got.ConnectionProfiles)
	}
	if got.APIKey != cfg.APIKey || got.BaseURL != cfg.BaseURL || got.Model != cfg.Model {
		t.Fatal("legacy root connection changed")
	}
}

func TestUpsertConnectionProfileCreatesRandomStableIDAndAutomaticName(t *testing.T) {
	cfg := NewDefaultConfig()
	draft := ConnectionProfile{
		BaseURL: "https://api.example.test/v1", APIProtocol: "openai_chat",
		EndpointPath: "/chat/completions", ResponseFormat: "text", Model: "model-a",
	}
	saved, err := UpsertConnectionProfile(&cfg, draft)
	if err != nil {
		t.Fatal(err)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, saved.ID); !matched {
		t.Fatalf("id=%q", saved.ID)
	}
	if saved.Name != "api.example.test · OpenAI Chat" {
		t.Fatalf("name=%q", saved.Name)
	}
	again, err := UpsertConnectionProfile(&cfg, saved)
	if err != nil || again.ID != saved.ID || len(cfg.ConnectionProfiles) != 1 {
		t.Fatalf("update changed identity: %#v err=%v", again, err)
	}
}

func TestAutomaticProfileNamesResolveCollisionsWithin64Bytes(t *testing.T) {
	cfg := NewDefaultConfig()
	longHost := strings.Repeat("a", 80) + ".example.test"
	first, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		BaseURL: "https://" + longHost + "/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		BaseURL: "https://" + longHost + "/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len([]byte(first.Name)) > MaxConnectionProfileNameBytes || len([]byte(second.Name)) > MaxConnectionProfileNameBytes {
		t.Fatalf("automatic names exceed limit: %q / %q", first.Name, second.Name)
	}
	if !strings.HasSuffix(first.Name, " · OpenAI Chat") || !strings.HasSuffix(second.Name, " · OpenAI Chat 2") {
		t.Fatalf("unexpected names: %q / %q", first.Name, second.Name)
	}
}

func TestManualProfileNameRejectsCaseInsensitiveDuplicateAndOverlongValue(t *testing.T) {
	cfg := NewDefaultConfig()
	base := ConnectionProfile{BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text"}
	base.Name = "Primary"
	if _, err := UpsertConnectionProfile(&cfg, base); err != nil {
		t.Fatal(err)
	}
	base.Name = " primary "
	if _, err := UpsertConnectionProfile(&cfg, base); profileErrorCode(err) != ProfileErrorNameDuplicate {
		t.Fatalf("duplicate err=%v", err)
	}
	base.Name = strings.Repeat("界", 22)
	if _, err := UpsertConnectionProfile(&cfg, base); profileErrorCode(err) != ProfileErrorInvalid {
		t.Fatalf("overlong err=%v", err)
	}
}

func TestConnectionProfileLimitIsTwenty(t *testing.T) {
	cfg := NewDefaultConfig()
	for i := 0; i < MaxConnectionProfiles; i++ {
		_, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
			Name: "profile-" + string(rune('A'+i)), BaseURL: "https://example.test/v1",
			APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "overflow", BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if profileErrorCode(err) != ProfileErrorLimitReached {
		t.Fatalf("limit err=%v", err)
	}
}

func TestSameURLProfilesCoexistByID(t *testing.T) {
	cfg := NewDefaultConfig()
	first, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "first", APIKey: "key-a", BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "second", APIKey: "key-b", BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || len(cfg.ConnectionProfiles) != 2 {
		t.Fatalf("profiles=%#v", cfg.ConnectionProfiles)
	}
}

func TestNormalizeConnectionProfilesRejectsMissingActiveID(t *testing.T) {
	cfg := NewDefaultConfig()
	saved, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "primary", BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg.ActiveConnectionProfileID = saved.ID + "00"
	if _, err := NormalizeConnectionProfiles(cfg); profileErrorCode(err) != ProfileErrorInvalid {
		t.Fatalf("err=%v", err)
	}
}

func TestApplyProfileMirrorsAllConnectionFieldsAndForcesDirectMode(t *testing.T) {
	cfg := NewDefaultConfig()
	profile := ConnectionProfile{
		ID: "0123456789abcdef0123456789abcdef", Name: "template", APIKey: "test-key",
		BaseURL: "https://example.test/generate", APIProtocol: "custom_json",
		ExtraHeaders: map[string]string{"X-Route": "a"}, ResponseFormat: "text",
		RequestTemplate: `{"messages":"{{openai_messages}}"}`, ResponseJSONPointer: "/answer",
		AuthHeader: "X-API-Key", AuthPrefix: "Token ", Model: "model-a",
	}
	ApplyProfileToActiveConnection(&cfg, profile)
	if cfg.ConnectionMode != ConnectionModeDirect || cfg.ActiveConnectionProfileID != profile.ID || cfg.APIKey != profile.APIKey ||
		cfg.BaseURL != profile.BaseURL || cfg.APIProtocol != profile.APIProtocol || cfg.RequestTemplate != profile.RequestTemplate ||
		cfg.ResponseJSONPointer != profile.ResponseJSONPointer || cfg.AuthHeader != profile.AuthHeader || cfg.AuthPrefix != profile.AuthPrefix || cfg.Model != profile.Model {
		t.Fatalf("mirror=%+v", cfg)
	}
	cfg.ExtraHeaders["X-Route"] = "mutated"
	if profile.ExtraHeaders["X-Route"] == "mutated" {
		t.Fatal("root mirror aliases profile headers")
	}
}

func TestFindAndNormalizeConnectionProfilesReturnDetachedHeaderMaps(t *testing.T) {
	cfg := NewDefaultConfig()
	saved, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "primary", BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions",
		ExtraHeaders: map[string]string{"X-Route": "a"}, ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	ApplyProfileToActiveConnection(&cfg, saved)
	normalized, err := NormalizeConnectionProfiles(cfg)
	if err != nil {
		t.Fatal(err)
	}
	found, ok := FindConnectionProfile(normalized, saved.ID)
	if !ok {
		t.Fatal("profile not found")
	}
	found.ExtraHeaders["X-Route"] = "mutated"
	if normalized.ConnectionProfiles[0].ExtraHeaders["X-Route"] == "mutated" {
		t.Fatal("find result aliases config")
	}
}

func TestDeleteConnectionProfileRejectsActiveLegacyAndMissingIDs(t *testing.T) {
	cfg := NewDefaultConfig()
	active, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "active", BaseURL: "https://a.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
	})
	if err != nil {
		t.Fatal(err)
	}
	ApplyProfileToActiveConnection(&cfg, active)
	if err := DeleteConnectionProfile(&cfg, active.ID); profileErrorCode(err) != ProfileErrorActiveDeleteForbidden {
		t.Fatalf("active err=%v", err)
	}
	if err := DeleteConnectionProfile(&cfg, LegacyConnectionProfileID); profileErrorCode(err) != ProfileErrorLegacyDeleteForbidden {
		t.Fatalf("legacy err=%v", err)
	}
	if err := DeleteConnectionProfile(&cfg, "ffffffffffffffffffffffffffffffff"); profileErrorCode(err) != ProfileErrorNotFound {
		t.Fatalf("missing err=%v", err)
	}
}

func TestDeleteConnectionProfileKeepsOrder(t *testing.T) {
	cfg := NewDefaultConfig()
	var saved []ConnectionProfile
	for _, name := range []string{"first", "second", "third"} {
		profile, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
			Name: name, BaseURL: "https://example.test/v1", APIProtocol: "openai_chat", EndpointPath: "/chat/completions", ResponseFormat: "text",
		})
		if err != nil {
			t.Fatal(err)
		}
		saved = append(saved, profile)
	}
	ApplyProfileToActiveConnection(&cfg, saved[0])
	if err := DeleteConnectionProfile(&cfg, saved[1].ID); err != nil {
		t.Fatal(err)
	}
	if got := []string{cfg.ConnectionProfiles[0].Name, cfg.ConnectionProfiles[1].Name}; !reflect.DeepEqual(got, []string{"first", "third"}) {
		t.Fatalf("order=%v", got)
	}
}

func TestCustomJSONProfileNormalizesFullRequestURL(t *testing.T) {
	cfg := NewDefaultConfig()
	saved, err := UpsertConnectionProfile(&cfg, ConnectionProfile{
		Name: "template", BaseURL: "https://example.test/generate/", APIProtocol: "custom_json", EndpointPath: "/ignored",
		RequestTemplate: `{"messages":"{{openai_messages}}"}`, ResponseJSONPointer: "/answer", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Model: "model-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.BaseURL != "https://example.test/generate/" || saved.EndpointPath != "" {
		t.Fatalf("profile=%+v", saved)
	}
}
