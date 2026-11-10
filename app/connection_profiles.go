package app

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
)

type ConnectionProfileSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AddressLabel string `json:"addressLabel"`
	APIProtocol  string `json:"apiProtocol"`
	Model        string `json:"model"`
	Active       bool   `json:"active"`
	APIKeySet    bool   `json:"apiKeySet"`
	TemplateSet  bool   `json:"templateSet"`
}

type ConnectionProfileDraft struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Connection ConnectionDraft `json:"connection"`
}

type ConnectionProfileErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ConnectionProfileListResult struct {
	Profiles []ConnectionProfileSummary `json:"profiles"`
	Error    *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type GetConnectionProfileResult struct {
	Success bool                       `json:"success"`
	Profile *ConnectionProfileDraft    `json:"profile,omitempty"`
	Error   *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type DeleteConnectionProfileResult struct {
	Success   bool                       `json:"success"`
	DeletedID string                     `json:"deletedId,omitempty"`
	Error     *ConnectionProfileErrorDTO `json:"error,omitempty"`
}

type SaveConnectionResult struct {
	Success    bool                      `json:"success"`
	Profile    *ConnectionProfileSummary `json:"profile,omitempty"`
	Connection *ConnectionSettingsView   `json:"connection,omitempty"`
	Error      *ProviderErrorDTO         `json:"error,omitempty"`
}

var connectionProfileMessages = map[string]string{
	config.ProfileErrorLoadFailed:            "连接记录读取失败，请重新打开设置后重试",
	config.ProfileErrorNotFound:              "连接记录不存在或已被删除",
	config.ProfileErrorInvalid:               "连接记录配置无效，请检查后重试",
	config.ProfileErrorLimitReached:          "最多只能保存 20 条连接记录",
	config.ProfileErrorNameDuplicate:         "连接记录名称已存在，请使用其他名称",
	config.ProfileErrorActiveDeleteForbidden: "活动连接不能直接删除，请先保存激活另一条记录",
	config.ProfileErrorLegacyDeleteForbidden: "当前旧连接保存为正式记录前不能删除",
	config.ProfileErrorSaveFailed:            "连接记录保存失败，原配置未修改",
	config.ProfileErrorDeleteFailed:          "连接记录删除失败，原配置未修改",
}

func connectionProfileError(code string) *ConnectionProfileErrorDTO {
	message, ok := connectionProfileMessages[code]
	if !ok {
		code = config.ProfileErrorInvalid
		message = connectionProfileMessages[code]
	}
	return &ConnectionProfileErrorDTO{Code: code, Message: message}
}

func connectionProfileErrorCode(err error, fallback string) string {
	var profileErr *config.ProfileError
	if errors.As(err, &profileErr) && profileErr != nil && profileErr.Code != "" {
		return profileErr.Code
	}
	return fallback
}

func connectionProfileSaveError(err error, fallback string) SaveConnectionResult {
	code := connectionProfileErrorCode(err, fallback)
	message, ok := connectionProfileMessages[code]
	if !ok {
		code = config.ProfileErrorInvalid
		message = connectionProfileMessages[code]
	}
	return SaveConnectionResult{Error: &ProviderErrorDTO{
		Code: code, Stage: string(llm.StageValidateConfig), Message: message,
	}}
}

func connectionAddressLabel(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" || (parsed.Scheme == "https" && port == "443") || (parsed.Scheme == "http" && port == "80") {
		return host
	}
	return net.JoinHostPort(host, port)
}

func connectionProfileSummary(profile config.ConnectionProfile, activeID string) ConnectionProfileSummary {
	return ConnectionProfileSummary{
		ID:           profile.ID,
		Name:         profile.Name,
		AddressLabel: connectionAddressLabel(profile.BaseURL),
		APIProtocol:  profile.APIProtocol,
		Model:        profile.Model,
		Active:       profile.ID == activeID,
		APIKeySet:    strings.TrimSpace(profile.APIKey) != "",
		TemplateSet:  strings.TrimSpace(profile.RequestTemplate) != "",
	}
}

func connectionDraftFromProfile(profile config.ConnectionProfile) ConnectionDraft {
	return ConnectionDraft{
		ConnectionMode:      config.ConnectionModeDirect,
		APIKey:              profile.APIKey,
		BaseURL:             profile.BaseURL,
		Model:               profile.Model,
		APIProtocol:         profile.APIProtocol,
		EndpointPath:        profile.EndpointPath,
		ExtraHeaders:        cloneHeaders(profile.ExtraHeaders),
		ResponseFormat:      profile.ResponseFormat,
		RequestTemplate:     profile.RequestTemplate,
		ResponseJSONPointer: profile.ResponseJSONPointer,
		AuthHeader:          profile.AuthHeader,
		AuthPrefix:          profile.AuthPrefix,
	}
}

func cloneHeaders(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func connectionProfileFromDraft(draft ConnectionProfileDraft, connection config.Config) config.ConnectionProfile {
	return config.ConnectionProfile{
		ID:                  draft.ID,
		Name:                draft.Name,
		APIKey:              connection.APIKey,
		BaseURL:             connection.BaseURL,
		APIProtocol:         connection.APIProtocol,
		EndpointPath:        connection.EndpointPath,
		ExtraHeaders:        cloneHeaders(connection.ExtraHeaders),
		ResponseFormat:      connection.ResponseFormat,
		RequestTemplate:     connection.RequestTemplate,
		ResponseJSONPointer: connection.ResponseJSONPointer,
		AuthHeader:          connection.AuthHeader,
		AuthPrefix:          connection.AuthPrefix,
		Model:               connection.Model,
	}
}

func (a *App) ListConnectionProfiles() ConnectionProfileListResult {
	if a == nil || a.configManager == nil {
		return ConnectionProfileListResult{Profiles: []ConnectionProfileSummary{}, Error: connectionProfileError(config.ProfileErrorLoadFailed)}
	}
	if err := a.configManager.Load(); err != nil {
		return ConnectionProfileListResult{Profiles: []ConnectionProfileSummary{}, Error: connectionProfileError(config.ProfileErrorLoadFailed)}
	}
	cfg := a.configManager.Get()
	if len(cfg.ConnectionProfiles) == 0 {
		legacy := config.ConnectionProfile{
			ID: config.LegacyConnectionProfileID, Name: "当前连接", APIKey: cfg.APIKey,
			BaseURL: cfg.BaseURL, APIProtocol: cfg.APIProtocol, EndpointPath: cfg.EndpointPath,
			ExtraHeaders: cfg.ExtraHeaders, ResponseFormat: cfg.ResponseFormat,
			RequestTemplate: cfg.RequestTemplate, ResponseJSONPointer: cfg.ResponseJSONPointer,
			AuthHeader: cfg.AuthHeader, AuthPrefix: cfg.AuthPrefix, Model: cfg.Model,
		}
		summary := connectionProfileSummary(legacy, config.LegacyConnectionProfileID)
		return ConnectionProfileListResult{Profiles: []ConnectionProfileSummary{summary}}
	}
	profiles := make([]ConnectionProfileSummary, 0, len(cfg.ConnectionProfiles))
	for _, profile := range cfg.ConnectionProfiles {
		profiles = append(profiles, connectionProfileSummary(profile, cfg.ActiveConnectionProfileID))
	}
	return ConnectionProfileListResult{Profiles: profiles}
}

func (a *App) GetConnectionProfile(id string) GetConnectionProfileResult {
	if a == nil || a.configManager == nil {
		return GetConnectionProfileResult{Error: connectionProfileError(config.ProfileErrorLoadFailed)}
	}
	if err := a.configManager.Load(); err != nil {
		return GetConnectionProfileResult{Error: connectionProfileError(config.ProfileErrorLoadFailed)}
	}
	cfg := a.configManager.Get()
	id = strings.TrimSpace(id)
	if id == config.LegacyConnectionProfileID {
		if len(cfg.ConnectionProfiles) != 0 {
			return GetConnectionProfileResult{Error: connectionProfileError(config.ProfileErrorNotFound)}
		}
		legacy := config.ConnectionProfile{
			ID: id, APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, APIProtocol: cfg.APIProtocol,
			EndpointPath: cfg.EndpointPath, ExtraHeaders: cfg.ExtraHeaders,
			ResponseFormat: cfg.ResponseFormat, RequestTemplate: cfg.RequestTemplate,
			ResponseJSONPointer: cfg.ResponseJSONPointer, AuthHeader: cfg.AuthHeader,
			AuthPrefix: cfg.AuthPrefix, Model: cfg.Model,
		}
		profile := ConnectionProfileDraft{ID: id, Name: "", Connection: connectionDraftFromProfile(legacy)}
		return GetConnectionProfileResult{Success: true, Profile: &profile}
	}
	profile, ok := config.FindConnectionProfile(cfg, id)
	if !ok {
		return GetConnectionProfileResult{Error: connectionProfileError(config.ProfileErrorNotFound)}
	}
	draft := ConnectionProfileDraft{ID: profile.ID, Name: profile.Name, Connection: connectionDraftFromProfile(profile)}
	return GetConnectionProfileResult{Success: true, Profile: &draft}
}

func (a *App) SaveConnectionSettings(draft ConnectionProfileDraft) SaveConnectionResult {
	if a == nil || a.configManager == nil {
		return connectionProfileSaveError(nil, config.ProfileErrorSaveFailed)
	}
	connection, err := draft.Connection.config().NormalizedConnectionSettings()
	if err != nil {
		return connectionProfileSaveError(err, config.ProfileErrorInvalid)
	}
	profileDraft := connectionProfileFromDraft(draft, connection)
	var savedProfile config.ConnectionProfile
	var patchErr error
	err = a.configManager.PatchValidated(func(cfg *config.Config) {
		savedProfile, patchErr = config.UpsertConnectionProfile(cfg, profileDraft)
		if patchErr == nil {
			config.ApplyProfileToActiveConnection(cfg, savedProfile)
		}
	}, func(candidate config.Config) error {
		if patchErr != nil {
			return patchErr
		}
		_, providerErr := llm.NewProvider(candidate)
		return providerErr
	})
	if err != nil {
		return connectionProfileSaveError(err, config.ProfileErrorSaveFailed)
	}
	cfg := a.configManager.Get()
	stored, ok := config.FindConnectionProfile(cfg, cfg.ActiveConnectionProfileID)
	if !ok {
		return connectionProfileSaveError(nil, config.ProfileErrorSaveFailed)
	}
	summary := connectionProfileSummary(stored, cfg.ActiveConnectionProfileID)
	view := connectionSettingsView(cfg)
	return SaveConnectionResult{Success: true, Profile: &summary, Connection: &view}
}

func (a *App) DeleteConnectionProfile(id string) DeleteConnectionProfileResult {
	if a == nil || a.configManager == nil {
		return DeleteConnectionProfileResult{Error: connectionProfileError(config.ProfileErrorDeleteFailed)}
	}
	var deleteErr error
	err := a.configManager.PatchValidated(func(cfg *config.Config) {
		deleteErr = config.DeleteConnectionProfile(cfg, id)
	}, func(config.Config) error {
		return deleteErr
	})
	if err != nil {
		code := connectionProfileErrorCode(err, config.ProfileErrorDeleteFailed)
		return DeleteConnectionProfileResult{Error: connectionProfileError(code)}
	}
	return DeleteConnectionProfileResult{Success: true, DeletedID: strings.TrimSpace(id)}
}
