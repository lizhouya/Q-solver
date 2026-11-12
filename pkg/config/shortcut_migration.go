package config

import (
	"Q-Solver/pkg/shortcut"
	"runtime"
	"sort"
)

func repairCorruptFunctionKeyShortcuts(cfg *Config) []string {
	return repairCorruptFunctionKeyShortcutsForOS(cfg, runtime.GOOS)
}

func repairCorruptFunctionKeyShortcutsForOS(cfg *Config, goos string) []string {
	if cfg == nil || goos != "windows" || len(cfg.Shortcuts) == 0 {
		return nil
	}

	owners := make(map[string][]string)
	for action, binding := range cfg.Shortcuts {
		if binding.ComboID != "" {
			owners[binding.ComboID] = append(owners[binding.ComboID], action)
		}
	}

	proposals := make(map[string]string)
	targets := make(map[string][]string)
	for action, binding := range cfg.Shortcuts {
		actualName, ok := shortcut.ReadableNameFromComboID(binding.ComboID)
		if !ok || actualName == binding.KeyName {
			continue
		}
		target, ok := shortcut.FunctionKeyComboID(binding.KeyName)
		if !ok || target == binding.ComboID {
			continue
		}
		proposals[action] = target
		targets[target] = append(targets[target], action)
	}

	valid := make(map[string]string)
	for action, target := range proposals {
		if len(targets[target]) == 1 {
			valid[action] = target
		}
	}

	for changed := true; changed; {
		changed = false
		for action, target := range valid {
			blocked := false
			for _, owner := range owners[target] {
				if owner == action {
					continue
				}
				ownerTarget, movingAway := valid[owner]
				if !movingAway || ownerTarget == target {
					blocked = true
					break
				}
			}
			if blocked {
				delete(valid, action)
				changed = true
			}
		}
	}

	repaired := make([]string, 0, len(valid))
	for action, target := range valid {
		binding := cfg.Shortcuts[action]
		binding.ComboID = target
		cfg.Shortcuts[action] = binding
		repaired = append(repaired, action)
	}
	sort.Strings(repaired)
	return repaired
}
