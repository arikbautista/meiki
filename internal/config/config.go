// Package config provides config.toml and state.json management.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// UIConfig holds user-interface tuning knobs.
type UIConfig struct {
	BriefMaxOpenTodos int `toml:"brief_max_open_todos"`
	OpenScanDays      int `toml:"open_scan_days"`
	StaleTriageDays   int `toml:"stale_triage_days"`
}

// Config is the top-level configuration structure parsed from config.toml.
type Config struct {
	UI UIConfig `toml:"ui"`
}

// defaults returns a Config populated with sensible default values.
func defaults() Config {
	return Config{
		UI: UIConfig{
			BriefMaxOpenTodos: 20,
			OpenScanDays:      30,
			StaleTriageDays:   3,
		},
	}
}

// ConfigDir returns the meiki configuration directory.
// It respects XDG_CONFIG_HOME if set; otherwise uses ~/.config/meiki.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "meiki")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "meiki")
}

// DataDir returns the meiki data directory.
// It respects XDG_DATA_HOME if set; otherwise uses ~/.local/share/meiki.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "meiki")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "meiki")
}

// LoadConfig reads config.toml from ConfigDir and returns a Config.
// If the file does not exist, defaults are returned without error.
// Unknown keys in the file are silently ignored (forward-compatible).
// Only keys present in the file override the defaults.
func LoadConfig() (Config, error) {
	cfg := defaults()

	configFile := filepath.Join(ConfigDir(), "config.toml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file is fine — use defaults.
			return cfg, nil
		}
		return cfg, err
	}

	// Decode into a temporary struct so only present keys are overridden.
	var partial struct {
		UI struct {
			BriefMaxOpenTodos *int `toml:"brief_max_open_todos"`
			OpenScanDays      *int `toml:"open_scan_days"`
			StaleTriageDays   *int `toml:"stale_triage_days"`
		} `toml:"ui"`
	}

	if _, err := toml.Decode(string(data), &partial); err != nil {
		return cfg, err
	}

	if partial.UI.BriefMaxOpenTodos != nil {
		cfg.UI.BriefMaxOpenTodos = *partial.UI.BriefMaxOpenTodos
	}
	if partial.UI.OpenScanDays != nil {
		cfg.UI.OpenScanDays = *partial.UI.OpenScanDays
	}
	if partial.UI.StaleTriageDays != nil {
		cfg.UI.StaleTriageDays = *partial.UI.StaleTriageDays
	}

	return cfg, nil
}

// EnsureDataDir creates the meiki data directory tree if it does not exist.
// It creates DataDir(), DataDir()/entries/, and DataDir()/reviews/.
// It is idempotent and safe to call on every command invocation.
func EnsureDataDir() error {
	base := DataDir()
	for _, dir := range []string{base, filepath.Join(base, "entries"), filepath.Join(base, "reviews")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
