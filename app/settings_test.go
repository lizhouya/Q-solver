package app

import (
	"os"
	"path/filepath"
	"testing"

	"Q-Solver/pkg/common"
	"Q-Solver/pkg/config"
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
	if err != nil || got.APIKey != "test-only" || got.Model != "test-model" {
		t.Fatal("GetSettings before Startup must load saved config")
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
