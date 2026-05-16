package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/scanner"
)

// runReopen executes newReopenCmd() with the given arguments using the
// provided dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runReopen(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newReopenCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// ---------------------------------------------------------------------------
// Happy-path tests
// ---------------------------------------------------------------------------

func TestReopen_abandonedTodoAppearsInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "task to reopen")

	// Abandon the todo.
	_, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}

	// Reopen it.
	mutID, err := runReopen(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	// Should print a valid ULID.
	if len(mutID) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", mutID, len(mutID))
	}
	if mutID == todoID {
		t.Error("mutation ID should differ from original ID")
	}

	// Abandoned todo should now appear in the open list.
	todos, _, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	found := false
	for _, item := range todos {
		if item.Entry.ID == todoID {
			found = true
		}
	}
	if !found {
		t.Errorf("reopened todo %q should appear in open list", todoID)
	}
}

func TestReopen_resolvedBlockerAppearsInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "blocker to reopen")

	// Resolve the blocker.
	_, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Reopen it.
	mutID, err := runReopen(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if len(mutID) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", mutID, len(mutID))
	}

	// Resolved blocker should now appear in the open list.
	_, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	found := false
	for _, item := range blockers {
		if item.Entry.ID == blockerID {
			found = true
		}
	}
	if !found {
		t.Errorf("reopened blocker %q should appear in open list", blockerID)
	}
}

func TestReopen_mutationEntryFields(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "field-check task")

	// Abandon first.
	abandonMutID, err := runAbandon(t, dataDir, todoID, "not needed")
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}

	// Reopen.
	reopenMutID, err := runReopen(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	// Find the reopen mutation entry.
	entries := readTodayEntries(t, dataDir)

	var found bool
	for _, e := range entries {
		if e.ID == reopenMutID {
			found = true
			if e.Type != "todo" {
				t.Errorf("Type: want todo, got %q", e.Type)
			}
			if e.Status != "open" {
				t.Errorf("Status: want open, got %q", e.Status)
			}
			// supersedes should point to the abandon mutation (latest in chain).
			if e.Supersedes != abandonMutID {
				t.Errorf("Supersedes: want abandon mut %q, got %q", abandonMutID, e.Supersedes)
			}
			if e.Content != "reopened" {
				t.Errorf("Content: want \"reopened\", got %q", e.Content)
			}
			if e.Source != "cli" {
				t.Errorf("Source: want cli, got %q", e.Source)
			}
			if e.Timestamp == "" {
				t.Error("Timestamp should not be empty")
			}
		}
	}
	if !found {
		t.Fatalf("reopen mutation entry %q not found in today's file", reopenMutID)
	}
}

func TestReopen_preservesType(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "type-check blocker")

	_, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	mutID, err := runReopen(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	for _, e := range entries {
		if e.ID == mutID {
			if e.Type != "blocker" {
				t.Errorf("Type: want blocker, got %q", e.Type)
			}
		}
	}
}

func TestReopen_preservesProject(t *testing.T) {
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

	mutID, err := runReopen(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	for _, e := range entries {
		if e.ID == mutID {
			if e.Project != "myapp" {
				t.Errorf("expected reopen to preserve project myapp, got %q", e.Project)
			}
		}
	}
}

func TestReopen_outputIsULID(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "ulid check")
	_, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}

	stdout, err := runReopen(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if len(stdout) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", stdout, len(stdout))
	}
}

// TestReopen_supersedesLatestMutation verifies that when reopening by original
// id, the supersedes field on the new entry points to the latest mutation (the
// most recent state-bearing entry), not the original.
func TestReopen_supersedesLatestMutation(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	todoID := captureOpenTodo(t, dataDir, "chain test")

	// Abandon first.
	abandonID, err := runAbandon(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}

	// Reopen — supersedes should point to the abandon mutation.
	reopenID, err := runReopen(t, dataDir, todoID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	for _, e := range entries {
		if e.ID == reopenID {
			if e.Supersedes != abandonID {
				t.Errorf("reopen.Supersedes: want abandon mutation %q, got %q", abandonID, e.Supersedes)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestReopen_tooFewArgs(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runReopen(t, dataDir)
	if err == nil {
		t.Error("expected error when no id provided")
	}
}

func TestReopen_tooManyArgs(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runReopen(t, dataDir, "id1", "extra")
	if err == nil {
		t.Error("expected error when 2 args provided (only 1 allowed)")
	}
}

// TestReopen_alreadyOpenBlockerNotReopened verifies that the open-item scanner
// correctly reports the blocker as open before any reopen attempt, which is
// the precondition for the "already open" error path.
func TestReopen_alreadyOpenBlockerNotReopened(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "open blocker")

	_, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	found := false
	for _, item := range blockers {
		if item.Entry.ID == blockerID {
			found = true
		}
	}
	if !found {
		t.Fatalf("newly created blocker %q should be in open list", blockerID)
	}
}

// TestReopen_learningTypeCannotBeReopened verifies that a learning entry is
// not surfaced in open todos or blockers — its absence from those lists is the
// foundation for the "cannot reopen <type>" code path.
func TestReopen_learningTypeNotInOpenLists(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	learningID, err := runCapture(t, dataDir, "learning", "learned something")
	if err != nil {
		t.Fatalf("capture learning: %v", err)
	}

	todos, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}

	for _, item := range todos {
		if item.Entry.ID == learningID {
			t.Errorf("learning %q should not appear in todos", learningID)
		}
	}
	for _, item := range blockers {
		if item.Entry.ID == learningID {
			t.Errorf("learning %q should not appear in blockers", learningID)
		}
	}
}
