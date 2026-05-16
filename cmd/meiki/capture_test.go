package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/entry"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runCapture executes newCaptureCmd() with the given arguments using the
// provided XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runCapture(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newCaptureCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	// SilenceUsage is already set; also silence errors on the command itself.
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// todayFile returns the path to today's JSONL file within dataDir.
func todayFile(dataDir string) string {
	now := time.Now().UTC()
	y := now.Format("2006")
	m := now.Format("01")
	d := now.Format("2006-01-02")
	return filepath.Join(dataDir, "meiki", "entries", y, m, d+".jsonl")
}

// readTodayEntries reads all entries from today's file in dataDir.
func readTodayEntries(t *testing.T, dataDir string) []entry.Entry {
	t.Helper()
	path := todayFile(dataDir)
	entries, err := entry.ReadEntriesFromPath(path)
	if err != nil {
		t.Fatalf("readTodayEntries: %v", err)
	}
	return entries
}

// ---------------------------------------------------------------------------
// Basic capture tests
// ---------------------------------------------------------------------------

func TestCapture_todo_basic(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runCapture(t, dataDir, "todo", "do the thing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should print a ULID (26 chars).
	id := stdout
	if len(id) != 26 {
		t.Fatalf("expected ULID (26 chars), got %q (len=%d)", id, len(id))
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ID != id {
		t.Errorf("entry ID %q != printed ID %q", e.ID, id)
	}
	if e.Type != "todo" {
		t.Errorf("expected type todo, got %q", e.Type)
	}
	if e.Content != "do the thing" {
		t.Errorf("expected content %q, got %q", "do the thing", e.Content)
	}
	if e.Status != "open" {
		t.Errorf("expected status open, got %q", e.Status)
	}
	if e.Priority != "this-week" {
		t.Errorf("expected default priority this-week, got %q", e.Priority)
	}
	if e.Source != "cli" {
		t.Errorf("expected source cli, got %q", e.Source)
	}
}

func TestCapture_achievement_basic(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "achievement", "shipped the feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Type != "achievement" {
		t.Errorf("expected type achievement, got %q", e.Type)
	}
	// Achievements should have no status.
	if e.Status != "" {
		t.Errorf("expected empty status for achievement, got %q", e.Status)
	}
}

func TestCapture_blocker_hasOpenStatus(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "blocker", "waiting on legal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != "open" {
		t.Errorf("expected status open for blocker, got %q", entries[0].Status)
	}
}

func TestCapture_learning_basic(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "learning", "Go interfaces are powerful")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if len(entries) != 1 || entries[0].Type != "learning" {
		t.Fatalf("expected 1 learning entry, got %v", entries)
	}
}

func TestCapture_idea_basic(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "idea", "add fuzzy search")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if len(entries) != 1 || entries[0].Type != "idea" {
		t.Fatalf("expected 1 idea entry, got %v", entries)
	}
}

// ---------------------------------------------------------------------------
// Flag tests
// ---------------------------------------------------------------------------

func TestCapture_project_flag(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "task", "--project", "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if entries[0].Project != "myapp" {
		t.Errorf("expected project myapp, got %q", entries[0].Project)
	}
}

func TestCapture_tags_flag(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "review PR", "--tags", "review,code,go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	e := entries[0]
	if len(e.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %v", e.Tags)
	}
	wantTags := []string{"review", "code", "go"}
	for i, tag := range wantTags {
		if e.Tags[i] != tag {
			t.Errorf("tag[%d]: got %q, want %q", i, e.Tags[i], tag)
		}
	}
}

func TestCapture_priority_flag(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "urgent task", "--priority", "tomorrow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if entries[0].Priority != "tomorrow" {
		t.Errorf("expected priority tomorrow, got %q", entries[0].Priority)
	}
}

func TestCapture_due_flag(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "deadline task", "--due", "2026-12-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if entries[0].Due != "2026-12-31" {
		t.Errorf("expected due 2026-12-31, got %q", entries[0].Due)
	}
}

func TestCapture_externalRef_flag(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "jira task", "--external-ref", "jira:ENG-1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := readTodayEntries(t, dataDir)
	if entries[0].ExternalRef != "jira:ENG-1234" {
		t.Errorf("expected external_ref jira:ENG-1234, got %q", entries[0].ExternalRef)
	}
}

// ---------------------------------------------------------------------------
// --closes validation
// ---------------------------------------------------------------------------

func TestCapture_closes_validOpenTodo(t *testing.T) {
	dataDir := t.TempDir()

	// Create a todo first.
	todoID, err := runCapture(t, dataDir, "todo", "do the thing")
	if err != nil {
		t.Fatalf("create todo: %v", err)
	}

	// Capture an achievement that closes it.
	_, err = runCapture(t, dataDir, "achievement", "did the thing", "--closes", todoID)
	if err != nil {
		t.Fatalf("capture achievement with --closes: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	achievement := entries[1]
	if achievement.Closes != todoID {
		t.Errorf("expected closes %q, got %q", todoID, achievement.Closes)
	}
}

func TestCapture_closes_nonExistentID(t *testing.T) {
	dataDir := t.TempDir()
	// No entries exist; this should fail.
	_, err := runCapture(t, dataDir, "achievement", "did the thing", "--closes", "01ABCDEFGHIJKLMNOPQRSTUVWX")
	if err == nil {
		t.Error("expected error for non-existent --closes id, got nil")
	}
}

// ---------------------------------------------------------------------------
// Type-specific flag validation
// ---------------------------------------------------------------------------

func TestCapture_priority_on_non_todo(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "learning", "something", "--priority", "tomorrow")
	if err == nil {
		t.Error("expected error: --priority on non-todo type")
	}
}

func TestCapture_due_on_non_todo(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "blocker", "stuck", "--due", "2026-12-31")
	if err == nil {
		t.Error("expected error: --due on non-todo type")
	}
}

func TestCapture_closes_on_non_achievement(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "task", "--closes", "01ABCDEFGHIJKLMNOPQRSTUVWX")
	if err == nil {
		t.Error("expected error: --closes on non-achievement type")
	}
}

// ---------------------------------------------------------------------------
// Invalid type
// ---------------------------------------------------------------------------

func TestCapture_invalidType(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "bogus", "content")
	if err == nil {
		t.Error("expected error for invalid type, got nil")
	}
}

// ---------------------------------------------------------------------------
// Project auto-detection
// ---------------------------------------------------------------------------

func TestCapture_projectAutoDetect(t *testing.T) {
	dataDir := t.TempDir()

	// Change cwd to a temp directory with a known basename.
	projectDir := filepath.Join(t.TempDir(), "myrepo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	_, err = runCapture(t, dataDir, "todo", "auto project task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if entries[0].Project != "myrepo" {
		t.Errorf("expected project myrepo (auto-detected), got %q", entries[0].Project)
	}
}

// ---------------------------------------------------------------------------
// Entry written to correct daily file
// ---------------------------------------------------------------------------

func TestCapture_writesToDailyFile(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "idea", "daily file test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := todayFile(dataDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected daily JSONL file at %s, but it does not exist", path)
	}
}

// ---------------------------------------------------------------------------
// Tags parsing edge cases
// ---------------------------------------------------------------------------

func TestParseTags(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b , c  ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"a,,b", []string{"a", "b"}},   // empty segments dropped
		{",leading", []string{"leading"}},
	}

	for _, tc := range tests {
		got := parseTags(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i, tag := range tc.want {
			if got[i] != tag {
				t.Errorf("parseTags(%q)[%d] = %q, want %q", tc.input, i, got[i], tag)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Output is valid ULID
// ---------------------------------------------------------------------------

func TestCapture_outputIsULID(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runCapture(t, dataDir, "idea", "test ulid output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ULID is 26 uppercase alphanumeric characters.
	if len(stdout) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", stdout, len(stdout))
	}
}

// ---------------------------------------------------------------------------
// Entry is valid JSON
// ---------------------------------------------------------------------------

func TestCapture_entryIsValidJSON(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runCapture(t, dataDir, "todo", "json check", "--tags", "a,b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := todayFile(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %q", i, line)
		}
	}
}
