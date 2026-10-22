package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
