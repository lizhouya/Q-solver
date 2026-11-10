package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	LegacyConnectionProfileID         = "legacy-current"
	MaxConnectionProfiles             = 20
	MaxConnectionProfileNameBytes     = 64
	ProfileErrorLoadFailed            = "profile_load_failed"
	ProfileErrorNotFound              = "profile_not_found"
	ProfileErrorInvalid               = "profile_invalid"
	ProfileErrorLimitReached          = "profile_limit_reached"
	ProfileErrorNameDuplicate         = "profile_name_duplicate"
	ProfileErrorActiveDeleteForbidden = "profile_active_delete_forbidden"
	ProfileErrorLegacyDeleteForbidden = "profile_legacy_delete_forbidden"
	ProfileErrorSaveFailed            = "profile_save_failed"
	ProfileErrorDeleteFailed          = "profile_delete_failed"
)

type ProfileError struct {
	Code string
}

func (e *ProfileError) Error() string {
	if e == nil {
		return ""
	}
	return e.Code
}

func profileError(code string) error {
	return &ProfileError{Code: code}
}

func profileErrorCode(err error) string {
	if typed, ok := err.(*ProfileError); ok && typed != nil {
		return typed.Code
	}
	return ""
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneConnectionProfile(source ConnectionProfile) ConnectionProfile {
	result := source
	result.ExtraHeaders = cloneStringMap(source.ExtraHeaders)
	return result
}

func profileFromConnectionConfig(cfg Config, id, name string) ConnectionProfile {
	return ConnectionProfile{
		ID:                  id,
		Name:                name,
		APIKey:              cfg.APIKey,
		BaseURL:             cfg.BaseURL,
		APIProtocol:         cfg.APIProtocol,
		EndpointPath:        cfg.EndpointPath,
		ExtraHeaders:        cloneStringMap(cfg.ExtraHeaders),
		ResponseFormat:      cfg.ResponseFormat,
		RequestTemplate:     cfg.RequestTemplate,
		ResponseJSONPointer: cfg.ResponseJSONPointer,
		AuthHeader:          cfg.AuthHeader,
		AuthPrefix:          cfg.AuthPrefix,
		Model:               cfg.Model,
	}
}

func connectionConfigFromProfile(profile ConnectionProfile) Config {
	return Config{
		ConnectionMode:      ConnectionModeDirect,
		APIKey:              profile.APIKey,
		BaseURL:             profile.BaseURL,
		APIProtocol:         profile.APIProtocol,
		EndpointPath:        profile.EndpointPath,
		ExtraHeaders:        cloneStringMap(profile.ExtraHeaders),
		ResponseFormat:      profile.ResponseFormat,
		RequestTemplate:     profile.RequestTemplate,
		ResponseJSONPointer: profile.ResponseJSONPointer,
		AuthHeader:          profile.AuthHeader,
		AuthPrefix:          profile.AuthPrefix,
		Model:               profile.Model,
	}
}

func normalizeConnectionProfile(profile ConnectionProfile) (ConnectionProfile, error) {
	normalized, err := normalizeConnectionFields(connectionConfigFromProfile(profile))
	if err != nil {
		return ConnectionProfile{}, profileError(ProfileErrorInvalid)
	}
	result := profileFromConnectionConfig(normalized, profile.ID, strings.TrimSpace(profile.Name))
	return result, nil
}

func validConnectionProfileID(id string) bool {
	if len(id) != 32 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validConnectionProfileName(name string) bool {
	return name != "" && len([]byte(name)) <= MaxConnectionProfileNameBytes
}

func duplicateConnectionProfileName(profiles []ConnectionProfile, name, exceptID string) bool {
	for _, existing := range profiles {
		if existing.ID != exceptID && strings.EqualFold(strings.TrimSpace(existing.Name), name) {
			return true
		}
	}
	return false
}

func truncateUTF8Bytes(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	for len([]byte(value)) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(value)
		if size <= 0 {
			return ""
		}
		value = value[:len(value)-size]
	}
	return value
}

func automaticConnectionProfileName(profile ConnectionProfile, profiles []ConnectionProfile) string {
	host := ""
	if parsed, err := url.Parse(profile.BaseURL); err == nil {
		host = parsed.Hostname()
	}
	if host == "" {
		host = "未命名连接"
	}
	protocolName := "OpenAI Chat"
	if profile.APIProtocol == "custom_json" {
		protocolName = "自定义模板"
	}
	for index := 1; ; index++ {
		number := ""
		if index > 1 {
			number = fmt.Sprintf(" %d", index)
		}
		suffix := " · " + protocolName + number
		candidateHost := truncateUTF8Bytes(host, MaxConnectionProfileNameBytes-len([]byte(suffix)))
		if candidateHost == "" {
			candidateHost = truncateUTF8Bytes("未命名连接", MaxConnectionProfileNameBytes-len([]byte(suffix)))
		}
		candidate := candidateHost + suffix
		if !duplicateConnectionProfileName(profiles, candidate, "") {
			return candidate
		}
	}
}

func newConnectionProfileID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func nextConnectionProfileID(profiles []ConnectionProfile) (string, error) {
	for attempts := 0; attempts < 4; attempts++ {
		id, err := newConnectionProfileID()
		if err != nil {
			return "", err
		}
		if _, exists := findConnectionProfileIndex(profiles, id); !exists {
			return id, nil
		}
	}
	return "", profileError(ProfileErrorSaveFailed)
}

func findConnectionProfileIndex(profiles []ConnectionProfile, id string) (int, bool) {
	for index, profile := range profiles {
		if profile.ID == id {
			return index, true
		}
	}
	return -1, false
}

func NormalizeConnectionProfiles(cfg Config) (Config, error) {
	cfg.ExtraHeaders = cloneStringMap(cfg.ExtraHeaders)
	if len(cfg.ConnectionProfiles) == 0 {
		if strings.TrimSpace(cfg.ActiveConnectionProfileID) != "" {
			return Config{}, profileError(ProfileErrorInvalid)
		}
		cfg.ActiveConnectionProfileID = ""
		cfg.ConnectionProfiles = []ConnectionProfile{}
		return cfg, nil
	}
	if len(cfg.ConnectionProfiles) > MaxConnectionProfiles {
		return Config{}, profileError(ProfileErrorLimitReached)
	}

	profiles := make([]ConnectionProfile, 0, len(cfg.ConnectionProfiles))
	for _, source := range cfg.ConnectionProfiles {
		if !validConnectionProfileID(source.ID) || source.ID == LegacyConnectionProfileID {
			return Config{}, profileError(ProfileErrorInvalid)
		}
		name := strings.TrimSpace(source.Name)
		if !validConnectionProfileName(name) || duplicateConnectionProfileName(profiles, name, "") {
			return Config{}, profileError(ProfileErrorInvalid)
		}
		source.Name = name
		normalized, err := normalizeConnectionProfile(source)
		if err != nil {
			return Config{}, err
		}
		profiles = append(profiles, normalized)
	}

	activeID := strings.TrimSpace(cfg.ActiveConnectionProfileID)
	activeIndex, ok := findConnectionProfileIndex(profiles, activeID)
	if !ok {
		return Config{}, profileError(ProfileErrorInvalid)
	}
	cfg.ConnectionProfiles = profiles
	ApplyProfileToActiveConnection(&cfg, profiles[activeIndex])
	return cfg, nil
}

func FindConnectionProfile(cfg Config, id string) (ConnectionProfile, bool) {
	index, ok := findConnectionProfileIndex(cfg.ConnectionProfiles, strings.TrimSpace(id))
	if !ok {
		return ConnectionProfile{}, false
	}
	return cloneConnectionProfile(cfg.ConnectionProfiles[index]), true
}

func UpsertConnectionProfile(cfg *Config, draft ConnectionProfile) (ConnectionProfile, error) {
	if cfg == nil {
		return ConnectionProfile{}, profileError(ProfileErrorInvalid)
	}
	profiles := make([]ConnectionProfile, len(cfg.ConnectionProfiles))
	for index, profile := range cfg.ConnectionProfiles {
		profiles[index] = cloneConnectionProfile(profile)
	}

	requestedID := strings.TrimSpace(draft.ID)
	isNew := requestedID == "" || requestedID == LegacyConnectionProfileID
	updateIndex := -1
	if isNew {
		if len(profiles) >= MaxConnectionProfiles {
			return ConnectionProfile{}, profileError(ProfileErrorLimitReached)
		}
		id, err := nextConnectionProfileID(profiles)
		if err != nil {
			return ConnectionProfile{}, err
		}
		draft.ID = id
	} else {
		if !validConnectionProfileID(requestedID) {
			return ConnectionProfile{}, profileError(ProfileErrorInvalid)
		}
		index, ok := findConnectionProfileIndex(profiles, requestedID)
		if !ok {
			return ConnectionProfile{}, profileError(ProfileErrorNotFound)
		}
		updateIndex = index
		draft.ID = requestedID
	}

	draft.Name = strings.TrimSpace(draft.Name)
	if draft.Name == "" {
		if !isNew {
			return ConnectionProfile{}, profileError(ProfileErrorInvalid)
		}
		normalizedForName, err := normalizeConnectionProfile(draft)
		if err != nil {
			return ConnectionProfile{}, err
		}
		draft = normalizedForName
		draft.Name = automaticConnectionProfileName(draft, profiles)
	} else {
		if !validConnectionProfileName(draft.Name) {
			return ConnectionProfile{}, profileError(ProfileErrorInvalid)
		}
		if duplicateConnectionProfileName(profiles, draft.Name, draft.ID) {
			return ConnectionProfile{}, profileError(ProfileErrorNameDuplicate)
		}
	}

	normalized, err := normalizeConnectionProfile(draft)
	if err != nil {
		return ConnectionProfile{}, err
	}
	if updateIndex >= 0 {
		profiles[updateIndex] = normalized
	} else {
		profiles = append(profiles, normalized)
	}
	cfg.ConnectionProfiles = profiles
	return cloneConnectionProfile(normalized), nil
}

func DeleteConnectionProfile(cfg *Config, id string) error {
	if cfg == nil {
		return profileError(ProfileErrorInvalid)
	}
	id = strings.TrimSpace(id)
	if id == LegacyConnectionProfileID {
		return profileError(ProfileErrorLegacyDeleteForbidden)
	}
	if id == cfg.ActiveConnectionProfileID {
		return profileError(ProfileErrorActiveDeleteForbidden)
	}
	index, ok := findConnectionProfileIndex(cfg.ConnectionProfiles, id)
	if !ok {
		return profileError(ProfileErrorNotFound)
	}
	profiles := make([]ConnectionProfile, 0, len(cfg.ConnectionProfiles)-1)
	for current, profile := range cfg.ConnectionProfiles {
		if current != index {
			profiles = append(profiles, cloneConnectionProfile(profile))
		}
	}
	cfg.ConnectionProfiles = profiles
	return nil
}

func ApplyProfileToActiveConnection(cfg *Config, profile ConnectionProfile) {
	if cfg == nil {
		return
	}
	cfg.ActiveConnectionProfileID = profile.ID
	cfg.ConnectionMode = ConnectionModeDirect
	cfg.APIKey = profile.APIKey
	cfg.BaseURL = profile.BaseURL
	cfg.APIProtocol = profile.APIProtocol
	cfg.EndpointPath = profile.EndpointPath
	cfg.ExtraHeaders = cloneStringMap(profile.ExtraHeaders)
	cfg.ResponseFormat = profile.ResponseFormat
	cfg.RequestTemplate = profile.RequestTemplate
	cfg.ResponseJSONPointer = profile.ResponseJSONPointer
	cfg.AuthHeader = profile.AuthHeader
	cfg.AuthPrefix = profile.AuthPrefix
	cfg.Model = profile.Model
}
