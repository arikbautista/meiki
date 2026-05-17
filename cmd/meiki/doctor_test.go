package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runDoctor(t *testing.T, dataDir, configDir string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	cmd := newDoctorCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), err
}

// ensureMeikiOnPath builds the meiki binary to a temp dir and prepends it to PATH.
func ensureMeikiOnPath(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	// Create a fake meiki binary (just needs to exist and be executable).
	fakeBin := filepath.Join(binDir, "meiki")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("create fake meiki binary: %v", err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func TestDoctor_AllHealthy(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()
	ensureMeikiOnPath(t)

	// Create the expected directory structure
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(filepath.Join(meikiData, "entries"), 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	// Create a valid state.json
	os.WriteFile(filepath.Join(meikiData, "state.json"), []byte(`{"last_brief_ts":"","last_review_ts":""}`), 0o644)

	output, err := runDoctor(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "All checks passed.") {
		t.Errorf("expected 'All checks passed.' in output, got:\n%s", output)
	}
	if strings.Contains(output, "✗") {
		t.Errorf("expected no failures in output, got:\n%s", output)
	}
}

func TestDoctor_MissingEntriesDir(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	// Create data dir but only reviews, not entries
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(meikiData, 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	output, err := runDoctor(t, dataDir, configDir)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("expected failure marker in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Entries directory: missing") {
		t.Errorf("expected entries missing message, got:\n%s", output)
	}
}

func TestDoctor_MalformedStateJSON(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	// Create full directory structure
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(filepath.Join(meikiData, "entries"), 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	// Write malformed state.json
	os.WriteFile(filepath.Join(meikiData, "state.json"), []byte(`{not valid json`), 0o644)

	output, err := runDoctor(t, dataDir, configDir)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("expected failure marker in output, got:\n%s", output)
	}
	if !strings.Contains(output, "State file: malformed JSON") {
		t.Errorf("expected malformed JSON message, got:\n%s", output)
	}
}

func TestDoctor_MalformedConfigTOML(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	// Create full directory structure
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(filepath.Join(meikiData, "entries"), 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	// Write malformed config.toml
	meikiConfig := filepath.Join(configDir, "meiki")
	os.MkdirAll(meikiConfig, 0o755)
	os.WriteFile(filepath.Join(meikiConfig, "config.toml"), []byte(`[ui\nnot valid toml`), 0o644)

	output, err := runDoctor(t, dataDir, configDir)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("expected failure marker in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Config file: parse error") {
		t.Errorf("expected config parse error message, got:\n%s", output)
	}
}

func TestDoctor_NoConfigFile(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()
	ensureMeikiOnPath(t)

	// Create full directory structure but no config file
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(filepath.Join(meikiData, "entries"), 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	output, err := runDoctor(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "Config file: not present (using defaults)") {
		t.Errorf("expected 'not present (using defaults)' message, got:\n%s", output)
	}
}

func TestDoctor_NoStateFile(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()
	ensureMeikiOnPath(t)

	// Create full directory structure but no state.json
	meikiData := filepath.Join(dataDir, "meiki")
	os.MkdirAll(filepath.Join(meikiData, "entries"), 0o755)
	os.MkdirAll(filepath.Join(meikiData, "reviews"), 0o755)

	output, err := runDoctor(t, dataDir, configDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v\noutput: %s", err, output)
	}
	if !strings.Contains(output, "State file: not present (OK)") {
		t.Errorf("expected 'not present (OK)' message, got:\n%s", output)
	}
}
