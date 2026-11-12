package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateUserConfigDirs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "HOME", "USERPROFILE"} {
		t.Setenv(name, root)
	}
	return root
}

func TestIsolateUserConfigDirsControlsPlatformConfigPath(t *testing.T) {
	root := isolateUserConfigDirs(t)
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(root, configDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("user config dir escaped isolated root: root=%q configDir=%q", root, configDir)
	}
}
