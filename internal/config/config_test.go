package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setEnv sets an environment variable and returns a cleanup function that
// restores the original value (or unsets it if it was not previously set).
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	original, wasSet := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if wasSet {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	})
}

// unsetEnv unsets an environment variable and returns a cleanup function that
// restores the original value if it was set.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, wasSet := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if wasSet {
			os.Setenv(key, original)
		}
	})
}

// --- ConfigDir tests ---

func TestConfigDir_Default(t *testing.T) {
	unsetEnv(t, "XDG_CONFIG_HOME")

	got := ConfigDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "meiki")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_XDGOverride(t *testing.T) {
	setEnv(t, "XDG_CONFIG_HOME", "/custom/config")

	got := ConfigDir()
	want := "/custom/config/meiki"
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

// --- DataDir tests ---

func TestDataDir_Default(t *testing.T) {
	unsetEnv(t, "XDG_DATA_HOME")

	got := DataDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "share", "meiki")
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDir_XDGOverride(t *testing.T) {
	setEnv(t, "XDG_DATA_HOME", "/custom/data")

	got := DataDir()
	want := "/custom/data/meiki"
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

// --- LoadConfig tests ---

// setupConfigDir creates a temporary directory that acts as XDG_CONFIG_HOME
// and writes an optional config.toml. Returns a cleanup function.
func setupConfigDir(t *testing.T, content string) {
	t.Helper()
	tmp := t.TempDir()
	setEnv(t, "XDG_CONFIG_HOME", tmp)

	if content != "" {
		meikiDir := filepath.Join(tmp, "meiki")
		if err := os.MkdirAll(meikiDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", meikiDir, err)
		}
		if err := os.WriteFile(filepath.Join(meikiDir, "config.toml"), []byte(content), 0o644); err != nil {
			t.Fatalf("write config.toml: %v", err)
		}
	}
}

func TestLoadConfig_Defaults_NoFile(t *testing.T) {
	// Point XDG_CONFIG_HOME to an empty temp dir so no config.toml exists.
	setupConfigDir(t, "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	d := defaults()
	if cfg.UI.BriefMaxOpenTodos != d.UI.BriefMaxOpenTodos {
		t.Errorf("BriefMaxOpenTodos = %d, want %d", cfg.UI.BriefMaxOpenTodos, d.UI.BriefMaxOpenTodos)
	}
	if cfg.UI.OpenScanDays != d.UI.OpenScanDays {
		t.Errorf("OpenScanDays = %d, want %d", cfg.UI.OpenScanDays, d.UI.OpenScanDays)
	}
	if cfg.UI.StaleTriageDays != d.UI.StaleTriageDays {
		t.Errorf("StaleTriageDays = %d, want %d", cfg.UI.StaleTriageDays, d.UI.StaleTriageDays)
	}
}

func TestLoadConfig_FullConfig(t *testing.T) {
	setupConfigDir(t, `
[ui]
brief_max_open_todos = 10
open_scan_days = 60
stale_triage_days = 7
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UI.BriefMaxOpenTodos != 10 {
		t.Errorf("BriefMaxOpenTodos = %d, want 10", cfg.UI.BriefMaxOpenTodos)
	}
	if cfg.UI.OpenScanDays != 60 {
		t.Errorf("OpenScanDays = %d, want 60", cfg.UI.OpenScanDays)
	}
	if cfg.UI.StaleTriageDays != 7 {
		t.Errorf("StaleTriageDays = %d, want 7", cfg.UI.StaleTriageDays)
	}
}

func TestLoadConfig_PartialConfig_OverridesOnlySpecifiedKeys(t *testing.T) {
	// Only override open_scan_days; the other two should remain at defaults.
	setupConfigDir(t, `
[ui]
open_scan_days = 45
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	d := defaults()
	if cfg.UI.OpenScanDays != 45 {
		t.Errorf("OpenScanDays = %d, want 45", cfg.UI.OpenScanDays)
	}
	if cfg.UI.BriefMaxOpenTodos != d.UI.BriefMaxOpenTodos {
		t.Errorf("BriefMaxOpenTodos = %d, want default %d", cfg.UI.BriefMaxOpenTodos, d.UI.BriefMaxOpenTodos)
	}
	if cfg.UI.StaleTriageDays != d.UI.StaleTriageDays {
		t.Errorf("StaleTriageDays = %d, want default %d", cfg.UI.StaleTriageDays, d.UI.StaleTriageDays)
	}
}

func TestLoadConfig_UnknownKeysIgnored(t *testing.T) {
	// Unknown keys should not cause an error.
	setupConfigDir(t, `
[ui]
brief_max_open_todos = 5
future_unknown_key = "hello"

[future_section]
another_key = 42
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v; want nil (unknown keys silently ignored)", err)
	}

	if cfg.UI.BriefMaxOpenTodos != 5 {
		t.Errorf("BriefMaxOpenTodos = %d, want 5", cfg.UI.BriefMaxOpenTodos)
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	setupConfigDir(t, "this is [not valid toml !!!")

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() error = nil, want parse error for invalid TOML")
	}
}

// --- EnsureDataDir tests ---

func TestEnsureDataDir_CreatesDirectoryTree(t *testing.T) {
	tmp := t.TempDir()
	setEnv(t, "XDG_DATA_HOME", tmp)

	if err := EnsureDataDir(); err != nil {
		t.Fatalf("EnsureDataDir() error = %v", err)
	}

	base := filepath.Join(tmp, "meiki")
	for _, sub := range []string{"", "entries", "reviews"} {
		dir := base
		if sub != "" {
			dir = filepath.Join(base, sub)
		}
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestEnsureDataDir_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	setEnv(t, "XDG_DATA_HOME", tmp)

	// Call twice — must not error.
	if err := EnsureDataDir(); err != nil {
		t.Fatalf("first EnsureDataDir() error = %v", err)
	}
	if err := EnsureDataDir(); err != nil {
		t.Fatalf("second EnsureDataDir() error = %v", err)
	}
}

func TestEnsureDataDir_ExistingFilesUntouched(t *testing.T) {
	tmp := t.TempDir()
	setEnv(t, "XDG_DATA_HOME", tmp)

	// Pre-create the directories and drop a sentinel file inside entries/.
	base := filepath.Join(tmp, "meiki")
	os.MkdirAll(filepath.Join(base, "entries"), 0o755)
	os.MkdirAll(filepath.Join(base, "reviews"), 0o755)
	sentinel := filepath.Join(base, "entries", "sentinel.jsonl")
	if err := os.WriteFile(sentinel, []byte("data"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	if err := EnsureDataDir(); err != nil {
		t.Fatalf("EnsureDataDir() error = %v", err)
	}

	// Sentinel must still exist with its original content.
	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("sentinel content = %q, want %q", got, "data")
	}
}
