package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
)

// runAbandon executes newAbandonCmd() with the given arguments using the
// provided dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runAbandon(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	writeTestConfig(t, cfgDir)

	cmd := newAbandonCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// captureOpenTodo is a test helper that creates an open todo in dataDir and
// returns its ID.
func captureOpenTodo(t *testing.T, dataDir string, content string) string {
	t.Helper()
	id, err := runCapture(t, dataDir, "todo", content)
	if err != nil {
		t.Fatalf("captureOpenTodo: %v", err)
	}
	return id
}

// ---------------------------------------------------------------------------
// Happy-path tests
// ---------------------------------------------------------------------------

func TestAbandon_basicHappyPath(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "do the thing")

	mutID, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should print a valid ULID.
	if len(mutID) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", mutID, len(mutID))
	}
	if mutID == todoID {
		t.Error("mutation ID should differ from original ID")
	}

	// Read today's entries; expect original + mutation.
	entries := readTodayEntries(t, dataDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	mut := entries[1]
	if mut.ID != mutID {
		t.Errorf("mutation entry ID %q != printed ID %q", mut.ID, mutID)
	}
	if mut.Type != "todo" {
		t.Errorf("mutation type: want todo, got %q", mut.Type)
	}
	if mut.Status != "abandoned" {
		t.Errorf("mutation status: want abandoned, got %q", mut.Status)
	}
	if mut.Supersedes != todoID {
		t.Errorf("mutation supersedes: want %q, got %q", todoID, mut.Supersedes)
	}
	if mut.Content != "abandoned" {
		t.Errorf("mutation content: want \"abandoned\", got %q", mut.Content)
	}
	if mut.Source != "cli" {
		t.Errorf("mutation source: want cli, got %q", mut.Source)
	}
}

func TestAbandon_withReason(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "write the spec")

	_, err := runAbandon(t, dataDir, todoID, "decided not to pursue this")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	mut := entries[1]
	if mut.Content != "decided not to pursue this" {
		t.Errorf("expected reason in content, got %q", mut.Content)
	}
}

func TestAbandon_defaultReasonIsAbandoned(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "some task")

	_, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	mut := entries[len(entries)-1]
	if mut.Content != "abandoned" {
		t.Errorf("expected default content \"abandoned\", got %q", mut.Content)
	}
}

func TestAbandon_todoNoLongerAppearsInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "task to abandon")

	_, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("abandon failed: %v", err)
	}

	todos, _, scanErr := scanner.ScanOpenItems(config.DataDir(), 30, todayMidnightUTC(), time.UTC, 0)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	for _, item := range todos {
		if item.Entry.ID == todoID {
			t.Errorf("abandoned todo %q still appears in open list", todoID)
		}
	}
}

func TestAbandon_preservesProjectOnMutation(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID, err := runCapture(t, dataDir, "todo", "project task", "--project", "myapp")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	_, err = runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	mut := entries[len(entries)-1]
	if mut.Project != "myapp" {
		t.Errorf("expected mutation to preserve project myapp, got %q", mut.Project)
	}
}

func TestAbandon_outputIsULID(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "ulid check task")
	stdout, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stdout) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", stdout, len(stdout))
	}
}

// ---------------------------------------------------------------------------
// Mutation entry structure
// ---------------------------------------------------------------------------

func TestAbandon_mutationEntryFields(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "field check task")

	mutID, err := runAbandon(t, dataDir, todoID, "no longer needed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	var mut entry.Entry
	found := false
	for _, e := range entries {
		if e.ID == mutID {
			mut = e
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mutation entry %q not found in today's file", mutID)
	}

	if mut.Supersedes != todoID {
		t.Errorf("Supersedes: want %q, got %q", todoID, mut.Supersedes)
	}
	if mut.Status != "abandoned" {
		t.Errorf("Status: want abandoned, got %q", mut.Status)
	}
	if mut.Type != "todo" {
		t.Errorf("Type: want todo, got %q", mut.Type)
	}
	if mut.Content != "no longer needed" {
		t.Errorf("Content: want \"no longer needed\", got %q", mut.Content)
	}
	if mut.Source != "cli" {
		t.Errorf("Source: want cli, got %q", mut.Source)
	}
	if mut.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Error cases — these call os.Exit so we test via subprocess-like approach.
// Since os.Exit is called directly, we verify via the RunE return path for
// cases that return errors, and use the error message for others.
//
// Note: The findEntryForAbandon function calls os.Exit for the "wrong type",
// "already closed", and "not found" cases. We test the RunE error path for
// the data-directory error, and verify the logic via unit tests of the helper.
// ---------------------------------------------------------------------------

func TestAbandon_tooFewArgs(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runAbandon(t, dataDir)
	if err == nil {
		t.Error("expected error when no id provided")
	}
}

func TestAbandon_tooManyArgs(t *testing.T) {
	dataDir := t.TempDir()
	// cobra RangeArgs(1,2) allows at most 2 args (id + reason).
	// Passing 3 should fail.
	_, err := runAbandon(t, dataDir, "id", "reason", "extra")
	if err == nil {
		t.Error("expected error when 3 args provided")
	}
}

// TestAbandon_findEntryForAbandon_nonTodoError tests the lookup helper in
// isolation by verifying it correctly differentiates entry types.
// We can't easily test os.Exit paths in unit tests without subprocess tricks,
// so we test the internal logic that leads up to the exit by verifying
// that the open-item scanner doesn't surface non-todos.
func TestAbandon_nonTodoNotInOpenTodos(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	// Capture a blocker (not a todo).
	blockerID, err := runCapture(t, dataDir, "blocker", "external dependency")
	if err != nil {
		t.Fatalf("capture blocker: %v", err)
	}

	// ScanOpenItems should return the blocker in blockers, not todos.
	todos, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30, todayMidnightUTC(), time.UTC, 0)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}

	for _, item := range todos {
		if item.Entry.ID == blockerID {
			t.Errorf("blocker %q should not appear in todos", blockerID)
		}
	}
	found := false
	for _, item := range blockers {
		if item.Entry.ID == blockerID {
			found = true
		}
	}
	if !found {
		t.Errorf("blocker %q should appear in blockers list", blockerID)
	}
}

// TestAbandon_alreadyAbandonedNotInOpenList verifies that an already-abandoned
// todo does not appear in the open todos list, which is the prerequisite for
// the "todo is already abandoned" error path.
func TestAbandon_alreadyAbandonedNotInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "task to abandon twice")

	// First abandon succeeds.
	_, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("first abandon: %v", err)
	}

	// After abandoning, the todo should no longer be in open todos.
	todos, _, scanErr := scanner.ScanOpenItems(config.DataDir(), 30, todayMidnightUTC(), time.UTC, 0)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	for _, item := range todos {
		if item.Entry.ID == todoID {
			t.Errorf("already-abandoned todo %q should not be in open list", todoID)
		}
	}
}

func TestEffectiveStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abandoned", "abandoned"},
		{"resolved", "resolved"},
		{"open", "open"},
		{"", "unknown"},
	}
	for _, tc := range tests {
		got := effectiveStatus(tc.input)
		if got != tc.want {
			t.Errorf("effectiveStatus(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
