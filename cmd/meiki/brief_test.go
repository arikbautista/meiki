package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runBrief executes newBriefCmd() with the given arguments using the provided
// dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runBrief(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	writeTestConfig(t, cfgDir)

	cmd := newBriefCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// writeBriefEntry writes a single entry to the JSONL file for the given date
// in XDG_DATA_HOME/meiki/entries/YYYY/MM/YYYY-MM-DD.jsonl.
func writeBriefEntry(t *testing.T, dataDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(dataDir, "meiki", "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeBriefEntry: %v", err)
	}
}

// writeBriefEntryForDate writes an entry with a status field (for open items).
func makeOpenEntry(id, entryType, content, project string, ts time.Time) entry.Entry {
	e := makeEntry(id, entryType, content, project, ts)
	e.Status = "open"
	return e
}

// ---------------------------------------------------------------------------
// Welcome message — no history
// ---------------------------------------------------------------------------

func TestBrief_noHistory_showsWelcome(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Welcome to meiki") {
		t.Errorf("expected welcome message, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Open todos appear in briefing
// ---------------------------------------------------------------------------

func TestBrief_showsOpenTodos(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeOpenEntry("TODOID0000000000000000001", "todo", "write the brief command", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	stdout, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "write the brief command") {
		t.Errorf("expected todo content in stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Open Todos") {
		t.Errorf("expected 'Open Todos' section, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Open blockers appear in briefing
// ---------------------------------------------------------------------------

func TestBrief_showsOpenBlockers(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeOpenEntry("BLKID00000000000000000001", "blocker", "CI pipeline is broken", "meiki", now)
	writeBriefEntry(t, tmpDir, now, e)

	stdout, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "CI pipeline is broken") {
		t.Errorf("expected blocker content in stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Open Blockers") {
		t.Errorf("expected 'Open Blockers' section, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// --json flag outputs valid JSON with required fields
// ---------------------------------------------------------------------------

func TestBrief_jsonFlag_validJSON(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeOpenEntry("TODOID0000000000000000001", "todo", "implement json output", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	stdout, err := runBrief(t, tmpDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, stdout)
	}

	// Check required top-level fields.
	requiredFields := []string{"review_summary", "open_todos", "open_blockers", "needs_triage"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("expected field %q in JSON output, not found\noutput:\n%s", field, stdout)
		}
	}
}

func TestBrief_jsonFlag_openTodosIncluded(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeOpenEntry("TODOID0000000000000000001", "todo", "json todo item", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	stdout, err := runBrief(t, tmpDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		OpenTodos []struct {
			Content string `json:"content"`
		} `json:"open_todos"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, stdout)
	}

	if len(result.OpenTodos) == 0 {
		t.Fatalf("expected open_todos to be non-empty, got:\n%s", stdout)
	}
	if result.OpenTodos[0].Content != "json todo item" {
		t.Errorf("expected content %q, got %q", "json todo item", result.OpenTodos[0].Content)
	}
}

func TestBrief_jsonFlag_emptySlicesNotNull(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	// Only a todo — no blockers or triage items.
	e := makeOpenEntry("TODOID0000000000000000001", "todo", "only todo", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	stdout, err := runBrief(t, tmpDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// open_blockers and needs_triage must be [] not null.
	if strings.Contains(stdout, `"open_blockers": null`) {
		t.Errorf("open_blockers should be [] not null:\n%s", stdout)
	}
	if strings.Contains(stdout, `"needs_triage": null`) {
		t.Errorf("needs_triage should be [] not null:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Debouncing: second run same day with no new entries outputs nothing
// ---------------------------------------------------------------------------

func TestBrief_debounced_noOutput(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	// Write an open todo so the first run produces output.
	e := makeOpenEntry("TODOID0000000000000000001", "todo", "debounce test todo", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	// First run — should produce output and update state.
	first, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if !strings.Contains(first, "debounce test todo") {
		t.Errorf("expected todo in first run output, got:\n%s", first)
	}

	// Second run — same day, no new entries — should be debounced (empty output).
	second, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if second != "" {
		t.Errorf("expected empty output on debounced second run, got:\n%s", second)
	}
}

// ---------------------------------------------------------------------------
// UpdateBriefTS is called after producing output
// ---------------------------------------------------------------------------

func TestBrief_updatesLastBriefTS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	now := time.Now().UTC()

	// Need some history so it's not just a welcome message.
	e := makeOpenEntry("TODOID0000000000000000001", "todo", "check ts update", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	_, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := config.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.LastBriefTS == "" {
		t.Error("expected last_brief_ts to be set in state.json, got empty string")
	}
}

// ---------------------------------------------------------------------------
// Exit code is 0 in all cases
// ---------------------------------------------------------------------------

func TestBrief_exitCodeZero_noHistory(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := runBrief(t, tmpDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) on fresh install, got: %v", err)
	}
}

func TestBrief_exitCodeZero_withHistory(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeOpenEntry("TODOID0000000000000000001", "todo", "exit code test", "meiki", now)
	e.Priority = "this-week"
	writeBriefEntry(t, tmpDir, now, e)

	_, err := runBrief(t, tmpDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0), got: %v", err)
	}
}

func TestBrief_exitCodeZero_debounced(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	now := time.Now().UTC()

	// Seed a last_brief_ts for today so debouncing kicks in immediately.
	state := config.State{
		LastBriefTS: now.UTC().Format(time.RFC3339),
	}
	if err := config.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	_, err := runBrief(t, tmpDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) when debounced, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Welcome message does NOT update last_brief_ts (not a real briefing)
// ---------------------------------------------------------------------------

func TestBrief_welcomeDoesNotUpdateTS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	_, err := runBrief(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := config.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.LastBriefTS != "" {
		t.Errorf("expected last_brief_ts to be empty after welcome, got %q", state.LastBriefTS)
	}
}
