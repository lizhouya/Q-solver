package config

import (
	"Q-Solver/pkg/common"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/shortcut"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ConfigManager struct {
	loaded      bool
	config      Config
	mu          sync.RWMutex
	updateMu    sync.Mutex
	configPath  string
	oldConfig   Config // 这是老配置
	subscribers []func(NewConfig Config, oldConfig Config)
}

func NewConfigManager() *ConfigManager {
	cm := &ConfigManager{
		config:      NewDefaultConfig(),
		oldConfig:   NewDefaultConfig(),
		subscribers: make([]func(NewConfig Config, oldConfig Config), 0),
	}
	cm.configPath = cm.getConfigPath()
	return cm
}

func appConfigDir() string {
	sysConfigDir, err := os.UserConfigDir()
	if err != nil {
		// 如果获取系统目录失败（极少情况），回退到当前目录
		sysConfigDir = "."
	}
	// 拼接项目名称目录
	appDir := filepath.Join(sysConfigDir, common.AppName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
	}
	return appDir
}

func (cm *ConfigManager) getConfigPath() string {
	fullPath := filepath.Join(appConfigDir(), "config")
	logger.Println("配置文件路径", fullPath)

	return fullPath
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.loaded {
		return nil
	}
	cfg := NewDefaultConfig()
	clearMobileDisplayConfigured := true
	startNextExamConfigured := true
	sendFastConfigured := true
	cancelSolveConfigured := true
	data, err := os.ReadFile(cm.configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("无法读取配置文件，原配置未修改，请检查文件访问权限后重试")
	}
	if err == nil {
		plain, decryptErr := decrypt(data)
		if decryptErr != nil {
			return fmt.Errorf("配置解密失败，原配置未修改，请检查配置文件后重试")
		}
		if json.Unmarshal(plain, &cfg) != nil {
			return fmt.Errorf("配置格式无效，原配置未修改，请检查配置文件后重试")
		}
		var persisted struct {
			Shortcuts    map[string]json.RawMessage `json:"shortcuts"`
			EndpointPath json.RawMessage            `json:"endpointPath"`
		}
		if json.Unmarshal(plain, &persisted) != nil {
			return fmt.Errorf("配置格式无效，原配置未修改，请检查配置文件后重试")
		}
		_, clearMobileDisplayConfigured = persisted.Shortcuts["clear_mobile_display"]
		_, startNextExamConfigured = persisted.Shortcuts["start_next_exam"]
		_, sendFastConfigured = persisted.Shortcuts["send_fast"]
		_, cancelSolveConfigured = persisted.Shortcuts["cancel_solve"]
		if len(persisted.EndpointPath) == 0 {
			cfg.EndpointPath = ""
		}
	}
	migrateClearMobileDisplayShortcut(&cfg, clearMobileDisplayConfigured)
	migrateStartNextExamShortcut(&cfg, startNextExamConfigured)
	defaultShortcuts := getDefaultShortcuts()
	migrateDefaultShortcut(&cfg, "send_fast", defaultShortcuts["send_fast"], sendFastConfigured)
	migrateDefaultShortcut(&cfg, "cancel_solve", defaultShortcuts["cancel_solve"], cancelSolveConfigured)
	normalized, normalizeErr := cfg.NormalizedConnectionSettings()
	if normalizeErr != nil {
		return fmt.Errorf("配置格式无效，原配置未修改，请检查配置文件后重试")
	}
	cfg = normalized
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置格式无效，原配置未修改，请检查配置文件后重试")
	}
	cm.config = cfg
	cm.loaded = true
	logger.Println("配置已加载")
	return nil
}

func (cm *ConfigManager) Save() error {
	cm.updateMu.Lock()
	defer cm.updateMu.Unlock()

	cm.mu.RLock()
	if !cm.loaded {
		cm.mu.RUnlock()
		return fmt.Errorf("配置尚未成功加载，已阻止保存以保护原配置")
	}
	candidate := cm.config
	cm.mu.RUnlock()

	return cm.saveSnapshot(candidate)
}

func (cm *ConfigManager) saveSnapshot(candidate Config) error {
	plain, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data, err := encrypt(plain)
	if err != nil {
		return fmt.Errorf("加密配置失败: %w", err)
	}
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Printf("配置已保存到: %s", cm.configPath)
	return nil
}

func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cloneConfigSnapshot(cm.config)
}

type connectionProfileUpdateMeta struct {
	ID   *string `json:"connectionProfileId"`
	Name *string `json:"connectionProfileName"`
}

// UpdateFromJSON 从前端 JSON 全量更新配置
func (cm *ConfigManager) UpdateFromJSON(jsonStr string) error {
	cm.updateMu.Lock()
	defer cm.updateMu.Unlock()

	if err := cm.Load(); err != nil {
		return err
	}

	cm.mu.RLock()
	oldConfigCopy := cloneConfigSnapshot(cm.config)
	cm.mu.RUnlock()

	newConfig := NewDefaultConfig()
	payload := []byte(jsonStr)
	if err := json.Unmarshal(payload, &newConfig); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}
	var persisted struct {
		EndpointPath json.RawMessage `json:"endpointPath"`
	}
	if err := json.Unmarshal(payload, &persisted); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}
	if len(persisted.EndpointPath) == 0 {
		newConfig.EndpointPath = ""
	}
	var meta connectionProfileUpdateMeta
	if err := json.Unmarshal(payload, &meta); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}

	normalizedConnection, err := normalizeConnectionFields(newConfig)
	if err != nil {
		return err
	}
	newConfig = normalizedConnection
	newConfig.ConnectionProfiles = cloneConfigSnapshot(oldConfigCopy).ConnectionProfiles
	newConfig.ActiveConnectionProfileID = oldConfigCopy.ActiveConnectionProfileID

	selectedID := oldConfigCopy.ActiveConnectionProfileID
	if meta.ID != nil {
		selectedID = strings.TrimSpace(*meta.ID)
	}
	if meta.ID == nil && selectedID == "" {
		selectedID = LegacyConnectionProfileID
	}
	if selectedID == LegacyConnectionProfileID && len(oldConfigCopy.ConnectionProfiles) > 0 {
		return profileError(ProfileErrorInvalid)
	}

	selectedName := ""
	if meta.Name != nil {
		selectedName = *meta.Name
	} else if existing, ok := FindConnectionProfile(oldConfigCopy, selectedID); ok {
		selectedName = existing.Name
	}
	profileDraft := profileFromConnectionConfig(newConfig, selectedID, selectedName)
	savedProfile, err := UpsertConnectionProfile(&newConfig, profileDraft)
	if err != nil {
		return err
	}
	ApplyProfileToActiveConnection(&newConfig, savedProfile)

	normalized, err := newConfig.NormalizedConnectionSettings()
	if err != nil {
		return err
	}
	newConfig = normalized
	if err := newConfig.Validate(); err != nil {
		return err
	}
	if err := cm.saveSnapshot(newConfig); err != nil {
		return err
	}

	cm.mu.Lock()
	cm.oldConfig = oldConfigCopy
	cm.config = newConfig
	configCopy := cloneConfigSnapshot(cm.config)
	subscribers := append([]func(NewConfig Config, oldConfig Config){}, cm.subscribers...)
	cm.mu.Unlock()

	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}
	return nil
}

// Patch 部分更新配置字段（避免全量序列化/反序列化的开销）
// patchFn 接收当前配置指针，直接修改需要变更的字段
func (cm *ConfigManager) Patch(patchFn func(cfg *Config)) error {
	return cm.PatchValidated(patchFn, nil)
}

// PatchValidated 在持久化和发布候选配置前执行附加验证。
func (cm *ConfigManager) PatchValidated(patchFn func(cfg *Config), validateFn func(candidate Config) error) error {
	cm.updateMu.Lock()
	defer cm.updateMu.Unlock()

	if err := cm.Load(); err != nil {
		return err
	}
	cm.mu.RLock()
	oldConfigCopy := cloneConfigSnapshot(cm.config)
	cm.mu.RUnlock()
	candidate := cloneConfigSnapshot(oldConfigCopy)
	patchFn(&candidate)
	normalized, err := candidate.NormalizedConnectionSettings()
	if err != nil {
		return err
	}
	candidate = normalized
	if err := candidate.Validate(); err != nil {
		return err
	}
	if validateFn != nil {
		if err := validateFn(cloneConfigSnapshot(candidate)); err != nil {
			return err
		}
	}
	if err := cm.saveSnapshot(candidate); err != nil {
		return err
	}

	cm.mu.Lock()
	cm.oldConfig = oldConfigCopy
	cm.config = candidate
	configCopy := cloneConfigSnapshot(cm.config)
	subscribers := append([]func(NewConfig Config, oldConfig Config){}, cm.subscribers...)
	cm.mu.Unlock()

	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}
	return nil
}

func cloneConfigSnapshot(source Config) Config {
	result := source
	if source.ExtraHeaders != nil {
		result.ExtraHeaders = make(map[string]string, len(source.ExtraHeaders))
		for key, value := range source.ExtraHeaders {
			result.ExtraHeaders[key] = value
		}
	}
	if source.Shortcuts != nil {
		result.Shortcuts = make(map[string]shortcut.KeyBinding, len(source.Shortcuts))
		for key, value := range source.Shortcuts {
			result.Shortcuts[key] = value
		}
	}
	if source.ConnectionProfiles != nil {
		result.ConnectionProfiles = make([]ConnectionProfile, len(source.ConnectionProfiles))
		for index, profile := range source.ConnectionProfiles {
			result.ConnectionProfiles[index] = cloneConnectionProfile(profile)
		}
	}
	return result
}

func (cm *ConfigManager) Subscribe(callback func(NewConfig Config, oldConfig Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.subscribers = append(cm.subscribers, callback)
}
