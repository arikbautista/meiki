package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
)

// runResolve executes newResolveCmd() with the given arguments using the
// provided dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runResolve(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newResolveCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// captureOpenBlocker is a test helper that creates an open blocker in dataDir
// and returns its ID.
func captureOpenBlocker(t *testing.T, dataDir string, content string) string {
	t.Helper()
	id, err := runCapture(t, dataDir, "blocker", content)
	if err != nil {
		t.Fatalf("captureOpenBlocker: %v", err)
	}
	return id
}

// ---------------------------------------------------------------------------
// Happy-path tests
// ---------------------------------------------------------------------------

func TestResolve_basicHappyPath(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "waiting on legal")

	mutID, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should print a valid ULID.
	if len(mutID) != 26 {
		t.Errorf("expected ULID (26 chars), got %q (len=%d)", mutID, len(mutID))
	}
	if mutID == blockerID {
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
	if mut.Type != "blocker" {
		t.Errorf("mutation type: want blocker, got %q", mut.Type)
	}
	if mut.Status != "resolved" {
		t.Errorf("mutation status: want resolved, got %q", mut.Status)
	}
	if mut.Supersedes != blockerID {
		t.Errorf("mutation supersedes: want %q, got %q", blockerID, mut.Supersedes)
	}
	if mut.Content != "resolved" {
		t.Errorf("mutation content: want \"resolved\", got %q", mut.Content)
	}
	if mut.Source != "cli" {
		t.Errorf("mutation source: want cli, got %q", mut.Source)
	}
}

func TestResolve_withDescription(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "waiting on API keys")

	_, err := runResolve(t, dataDir, blockerID, "vendor provided access")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	mut := entries[1]
	if mut.Content != "vendor provided access" {
		t.Errorf("expected description in content, got %q", mut.Content)
	}
}

func TestResolve_defaultContentIsResolved(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "some blocker")

	_, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	mut := entries[len(entries)-1]
	if mut.Content != "resolved" {
		t.Errorf("expected default content \"resolved\", got %q", mut.Content)
	}
}

func TestResolve_blockerNoLongerAppearsInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "blocker to resolve")

	_, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	_, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	for _, item := range blockers {
		if item.Entry.ID == blockerID {
			t.Errorf("resolved blocker %q still appears in open list", blockerID)
		}
	}
}

func TestResolve_preservesProjectOnMutation(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID, err := runCapture(t, dataDir, "blocker", "project blocker", "--project", "myapp")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	_, err = runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	entries := readTodayEntries(t, dataDir)
	mut := entries[len(entries)-1]
	if mut.Project != "myapp" {
		t.Errorf("expected mutation to preserve project myapp, got %q", mut.Project)
	}
}

func TestResolve_outputIsULID(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "ulid check blocker")
	stdout, err := runResolve(t, dataDir, blockerID)
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

func TestResolve_mutationEntryFields(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "field check blocker")

	mutID, err := runResolve(t, dataDir, blockerID, "legal team approved")
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

	if mut.Supersedes != blockerID {
		t.Errorf("Supersedes: want %q, got %q", blockerID, mut.Supersedes)
	}
	if mut.Status != "resolved" {
		t.Errorf("Status: want resolved, got %q", mut.Status)
	}
	if mut.Type != "blocker" {
		t.Errorf("Type: want blocker, got %q", mut.Type)
	}
	if mut.Content != "legal team approved" {
		t.Errorf("Content: want \"legal team approved\", got %q", mut.Content)
	}
	if mut.Source != "cli" {
		t.Errorf("Source: want cli, got %q", mut.Source)
	}
	if mut.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestResolve_tooFewArgs(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runResolve(t, dataDir)
	if err == nil {
		t.Error("expected error when no id provided")
	}
}

func TestResolve_tooManyArgs(t *testing.T) {
	dataDir := t.TempDir()
	// cobra RangeArgs(1,2) allows at most 2 args (id + how).
	// Passing 3 should fail.
	_, err := runResolve(t, dataDir, "id", "how", "extra")
	if err == nil {
		t.Error("expected error when 3 args provided")
	}
}

// TestResolve_nonBlockerNotInOpenBlockers verifies that the scanner correctly
// separates todos and blockers — a todo should not appear in open blockers.
func TestResolve_nonBlockerNotInOpenBlockers(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	// Capture a todo (not a blocker).
	todoID, err := runCapture(t, dataDir, "todo", "do the thing")
	if err != nil {
		t.Fatalf("capture todo: %v", err)
	}

	// ScanOpenItems should return the todo in todos, not blockers.
	todos, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}

	for _, item := range blockers {
		if item.Entry.ID == todoID {
			t.Errorf("todo %q should not appear in blockers", todoID)
		}
	}
	found := false
	for _, item := range todos {
		if item.Entry.ID == todoID {
			found = true
		}
	}
	if !found {
		t.Errorf("todo %q should appear in todos list", todoID)
	}
}

// TestResolve_alreadyResolvedNotInOpenList verifies that an already-resolved
// blocker does not appear in the open blockers list, which is the prerequisite
// for the "blocker is already resolved" error path.
func TestResolve_alreadyResolvedNotInOpenList(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	blockerID := captureOpenBlocker(t, dataDir, "blocker to resolve twice")

	// First resolve succeeds.
	_, err := runResolve(t, dataDir, blockerID)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}

	// After resolving, the blocker should no longer be in open blockers.
	_, blockers, scanErr := scanner.ScanOpenItems(config.DataDir(), 30)
	if scanErr != nil {
		t.Fatalf("ScanOpenItems: %v", scanErr)
	}
	for _, item := range blockers {
		if item.Entry.ID == blockerID {
			t.Errorf("already-resolved blocker %q should not be in open list", blockerID)
		}
	}
}
