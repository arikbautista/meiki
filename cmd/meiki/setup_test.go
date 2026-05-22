package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runSetup executes newSetupCmd() using the provided tmpDir for XDG dirs.
// Returns stdout output and any error.
func runSetup(t *testing.T, dataDir, configDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	cmd := newSetupCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// ---------------------------------------------------------------------------
// Directories are created
// ---------------------------------------------------------------------------

func TestSetup_createsDataDirectories(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	_, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Data directories should exist.
	for _, sub := range []string{"", "entries", "reviews"} {
		dir := filepath.Join(dataDir, "meiki", sub)
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("expected data directory %s to exist", dir)
		} else if err != nil {
			t.Errorf("stat %s: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestSetup_createsConfigDirectory(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	_, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := filepath.Join(configDir, "meiki")
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("expected config directory %s to exist", dir)
	} else if err != nil {
		t.Errorf("stat %s: %v", dir, err)
	} else if !info.IsDir() {
		t.Errorf("expected %s to be a directory", dir)
	}
}

// ---------------------------------------------------------------------------
// Output contains MEIKI.md content
// ---------------------------------------------------------------------------

func TestSetup_outputContainsMeikiMD(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	stdout, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for key MEIKI.md content markers.
	expectedFragments := []string{
		"MEIKI.md",
		"Session start",
		"meiki brief --json",
		"meiki capture achievement",
		"meiki abandon",
		"Anti-Patterns",
	}
	for _, frag := range expectedFragments {
		if !strings.Contains(stdout, frag) {
			t.Errorf("expected output to contain %q", frag)
		}
	}
}

// ---------------------------------------------------------------------------
// Output contains stop hook snippet
// ---------------------------------------------------------------------------

func TestSetup_outputContainsStopHook(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	stdout, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFragments := []string{
		`"hooks"`,
		`"Stop"`,
		`meiki review --silent`,
		"settings.json",
		"Optional if MEIKI.md",
	}
	for _, frag := range expectedFragments {
		if !strings.Contains(stdout, frag) {
			t.Errorf("expected output to contain %q", frag)
		}
	}
}

// ---------------------------------------------------------------------------
// Idempotent — re-running produces identical output
// ---------------------------------------------------------------------------

func TestSetup_idempotent(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	stdout1, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}

	stdout2, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}

	if stdout1 != stdout2 {
		t.Errorf("setup is not idempotent:\nfirst run:\n%s\nsecond run:\n%s", stdout1, stdout2)
	}
}

// ---------------------------------------------------------------------------
// Exit code is 0
// ---------------------------------------------------------------------------

func TestSetup_exitCodeZero(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	_, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Output contains doctor reminder
// ---------------------------------------------------------------------------

func TestSetup_outputContainsDoctorReminder(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	stdout, err := runSetup(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "meiki doctor") {
		t.Error("expected output to contain 'meiki doctor' reminder")
	}
}
