package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/entry"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// writeEntries writes a slice of Entry values to the appropriate JSONL path
// within dataDir for the given date.
func writeEntries(t *testing.T, dataDir string, date time.Time, entries []entry.Entry) {
	t.Helper()
	path := entryFilePath(dataDir, date)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()
	for _, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal entry: %v", err)
		}
		b = append(b, '\n')
		if _, err := f.Write(b); err != nil {
			t.Fatalf("write entry: %v", err)
		}
	}
}

// makeEntry creates an Entry with the given type and content. id must be
// a valid non-empty string (ULIDs are used in production; plain strings work
// fine in tests since we control all IDs).
func makeEntry(id, typ, content, status, supersedes, closes string) entry.Entry {
	return entry.Entry{
		ID:         id,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Type:       typ,
		Content:    content,
		Status:     status,
		Supersedes: supersedes,
		Closes:     closes,
	}
}

// makeEntryWithTime is like makeEntry but allows setting a specific timestamp
// so AgeDays can be tested.
func makeEntryWithTime(id, typ, content, status, supersedes, closes string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:         id,
		Timestamp:  ts.UTC().Format(time.RFC3339),
		Type:       typ,
		Content:    content,
		Status:     status,
		Supersedes: supersedes,
		Closes:     closes,
	}
}

// today returns today's date at midnight UTC.
func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestScanOpenItems_simpleOpen verifies that a single open todo is returned.
func TestScanOpenItems_simpleOpen(t *testing.T) {
	dir := t.TempDir()

	e := makeEntry("todo-001", "todo", "write unit tests", "open", "", "")
	writeEntries(t, dir, today(), []entry.Entry{e})

	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(blockers) != 0 {
		t.Errorf("expected 0 blockers, got %d", len(blockers))
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Entry.ID != "todo-001" {
		t.Errorf("unexpected todo ID: %q", todos[0].Entry.ID)
	}
	if todos[0].LatestState.Status != "open" {
		t.Errorf("expected status open, got %q", todos[0].LatestState.Status)
	}
}

// TestScanOpenItems_openBlocker verifies that an open blocker is returned in
// the blockers slice, not todos.
func TestScanOpenItems_openBlocker(t *testing.T) {
	dir := t.TempDir()

	e := makeEntry("blk-001", "blocker", "CI is broken", "open", "", "")
	writeEntries(t, dir, today(), []entry.Entry{e})

	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos, got %d", len(todos))
	}
	if len(blockers) != 1 {
		t.Fatalf("expected 1 blocker, got %d", len(blockers))
	}
	if blockers[0].Entry.ID != "blk-001" {
		t.Errorf("unexpected blocker ID: %q", blockers[0].Entry.ID)
	}
}

// TestScanOpenItems_closedViaAchievement verifies that a todo referenced by
// an achievement's `closes` field is not returned as open.
func TestScanOpenItems_closedViaAchievement(t *testing.T) {
	dir := t.TempDir()

	todo := makeEntry("todo-002", "todo", "deploy v1", "open", "", "")
	ach := makeEntry("ach-001", "achievement", "deployed v1 successfully", "", "", "todo-002")
	writeEntries(t, dir, today(), []entry.Entry{todo, ach})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 open todos (closed by achievement), got %d", len(todos))
	}
}

// TestScanOpenItems_abandonedTodo verifies that a todo with a mutation of
// status "abandoned" is not returned as open.
func TestScanOpenItems_abandonedTodo(t *testing.T) {
	dir := t.TempDir()

	orig := makeEntry("todo-003", "todo", "refactor auth", "open", "", "")
	mutation := makeEntry("todo-003-v2", "todo", "refactor auth", "abandoned", "todo-003", "")
	writeEntries(t, dir, today(), []entry.Entry{orig, mutation})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos (abandoned), got %d: %+v", len(todos), todos)
	}
}

// TestScanOpenItems_resolvedBlocker verifies that a blocker with a mutation of
// status "resolved" is not returned as open.
func TestScanOpenItems_resolvedBlocker(t *testing.T) {
	dir := t.TempDir()

	orig := makeEntry("blk-002", "blocker", "third-party API down", "open", "", "")
	mutation := makeEntry("blk-002-v2", "blocker", "third-party API down", "resolved", "blk-002", "")
	writeEntries(t, dir, today(), []entry.Entry{orig, mutation})

	_, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(blockers) != 0 {
		t.Errorf("expected 0 blockers (resolved), got %d", len(blockers))
	}
}

// TestScanOpenItems_reopenAfterClose verifies that an item closed then reopened
// appears as open (the latest state wins).
func TestScanOpenItems_reopenAfterClose(t *testing.T) {
	dir := t.TempDir()

	orig := makeEntry("todo-004", "todo", "write docs", "open", "", "")
	closed := makeEntry("todo-004-v2", "todo", "write docs", "abandoned", "todo-004", "")
	reopened := makeEntry("todo-004-v3", "todo", "write docs", "open", "todo-004-v2", "")
	writeEntries(t, dir, today(), []entry.Entry{orig, closed, reopened})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo (reopened), got %d", len(todos))
	}
	if todos[0].LatestState.ID != "todo-004-v3" {
		t.Errorf("expected latest state ID todo-004-v3, got %q", todos[0].LatestState.ID)
	}
	if todos[0].LatestState.Status != "open" {
		t.Errorf("expected status open, got %q", todos[0].LatestState.Status)
	}
}

// TestScanOpenItems_multiStepChain verifies that a 3-step supersedes chain
// (A → B → C) uses C's status.
func TestScanOpenItems_multiStepChain(t *testing.T) {
	dir := t.TempDir()

	a := makeEntry("todo-A", "todo", "original", "open", "", "")
	b := makeEntry("todo-B", "todo", "first mutation", "abandoned", "todo-A", "")
	c := makeEntry("todo-C", "todo", "second mutation (reopen)", "open", "todo-B", "")
	writeEntries(t, dir, today(), []entry.Entry{a, b, c})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 open todo (multi-step chain), got %d", len(todos))
	}
	if todos[0].Entry.ID != "todo-A" {
		t.Errorf("expected original Entry.ID todo-A, got %q", todos[0].Entry.ID)
	}
	if todos[0].LatestState.ID != "todo-C" {
		t.Errorf("expected LatestState.ID todo-C, got %q", todos[0].LatestState.ID)
	}
}

// TestScanOpenItems_missingFiles verifies that missing JSONL files in the scan
// range are silently skipped.
func TestScanOpenItems_missingFiles(t *testing.T) {
	dir := t.TempDir()
	// Write an entry for today only; the other 29 days have no files.
	e := makeEntry("todo-005", "todo", "something", "open", "", "")
	writeEntries(t, dir, today(), []entry.Entry{e})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems should not error on missing files: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("expected 1 todo, got %d", len(todos))
	}
}

// TestScanOpenItems_emptyDir verifies no error and empty results when the
// dataDir has no entries at all.
func TestScanOpenItems_emptyDir(t *testing.T) {
	dir := t.TempDir()
	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 0 || len(blockers) != 0 {
		t.Errorf("expected empty results, got todos=%d blockers=%d", len(todos), len(blockers))
	}
}

// TestScanOpenItems_respectsScanDays verifies that entries older than scanDays
// are not included.
func TestScanOpenItems_respectsScanDays(t *testing.T) {
	dir := t.TempDir()

	// Write an entry 5 days ago.
	fiveDaysAgo := today().AddDate(0, 0, -5)
	old := makeEntry("todo-old", "todo", "too old", "open", "", "")
	writeEntries(t, dir, fiveDaysAgo, []entry.Entry{old})

	// Write an entry today.
	recent := makeEntry("todo-recent", "todo", "recent", "open", "", "")
	writeEntries(t, dir, today(), []entry.Entry{recent})

	// Scan only 3 days back — should miss the 5-day-old entry.
	todos, _, err := ScanOpenItems(dir, 3)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo with scanDays=3, got %d", len(todos))
	}
	if todos[0].Entry.ID != "todo-recent" {
		t.Errorf("expected recent todo, got %q", todos[0].Entry.ID)
	}
}

// TestScanOpenItems_ageDays verifies that AgeDays reflects the time since the
// original entry was captured.
func TestScanOpenItems_ageDays(t *testing.T) {
	dir := t.TempDir()

	threeDaysAgo := today().AddDate(0, 0, -3)
	e := makeEntryWithTime("todo-aged", "todo", "old task", "open", "", "", threeDaysAgo)
	writeEntries(t, dir, threeDaysAgo, []entry.Entry{e})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].AgeDays != 3 {
		t.Errorf("expected AgeDays=3, got %d", todos[0].AgeDays)
	}
}

// TestScanOpenItems_nonStatusEntriesIgnored verifies that entries of type
// "achievement", "learning", and "idea" are not returned as open items.
func TestScanOpenItems_nonStatusEntriesIgnored(t *testing.T) {
	dir := t.TempDir()

	ach := makeEntry("ach-x", "achievement", "shipped!", "", "", "")
	lrn := makeEntry("lrn-x", "learning", "learned go", "", "", "")
	idea := makeEntry("idea-x", "idea", "new feature", "", "", "")
	writeEntries(t, dir, today(), []entry.Entry{ach, lrn, idea})

	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 || len(blockers) != 0 {
		t.Errorf("expected no open items for non-todo/blocker entries, got todos=%d blockers=%d", len(todos), len(blockers))
	}
}

// TestScanOpenItems_multipleItemsMixed verifies mixed scenarios in one scan.
func TestScanOpenItems_multipleItemsMixed(t *testing.T) {
	dir := t.TempDir()

	// Two open todos.
	t1 := makeEntry("t1", "todo", "task one", "open", "", "")
	t2 := makeEntry("t2", "todo", "task two", "open", "", "")
	// One abandoned todo.
	t3 := makeEntry("t3", "todo", "task three", "open", "", "")
	t3mut := makeEntry("t3-v2", "todo", "task three", "abandoned", "t3", "")
	// One open blocker.
	b1 := makeEntry("b1", "blocker", "block one", "open", "", "")
	// One todo closed by achievement.
	t4 := makeEntry("t4", "todo", "task four", "open", "", "")
	ach := makeEntry("a1", "achievement", "done four", "", "", "t4")

	writeEntries(t, dir, today(), []entry.Entry{t1, t2, t3, t3mut, b1, t4, ach})

	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 open todos, got %d: %+v", len(todos), func() []string {
			var ids []string
			for _, item := range todos {
				ids = append(ids, item.Entry.ID)
			}
			return ids
		}())
	}
	if len(blockers) != 1 {
		t.Errorf("expected 1 open blocker, got %d", len(blockers))
	}
}

// TestScanOpenItems_entriesAcrossMultipleDays verifies items spread across
// multiple days are all collected correctly.
func TestScanOpenItems_entriesAcrossMultipleDays(t *testing.T) {
	dir := t.TempDir()

	day1 := today().AddDate(0, 0, -10)
	day2 := today().AddDate(0, 0, -5)
	day3 := today()

	e1 := makeEntryWithTime("t-d1", "todo", "from day1", "open", "", "", day1)
	e2 := makeEntryWithTime("t-d2", "todo", "from day2", "open", "", "", day2)
	e3 := makeEntryWithTime("t-d3", "todo", "from day3", "open", "", "", day3)

	writeEntries(t, dir, day1, []entry.Entry{e1})
	writeEntries(t, dir, day2, []entry.Entry{e2})
	writeEntries(t, dir, day3, []entry.Entry{e3})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 3 {
		t.Errorf("expected 3 todos across days, got %d", len(todos))
	}
}

// TestScanOpenItems_todoWithNoStatus verifies that a todo entry with no status
// field is not returned as open (status must explicitly be "open").
func TestScanOpenItems_todoWithNoStatus(t *testing.T) {
	dir := t.TempDir()

	e := makeEntry("todo-nostatus", "todo", "no status set", "", "", "")
	writeEntries(t, dir, today(), []entry.Entry{e})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos (no status), got %d", len(todos))
	}
}

// TestScanOpenItems_defaultScanDays verifies that scanDays <= 0 defaults to 30.
func TestScanOpenItems_defaultScanDays(t *testing.T) {
	dir := t.TempDir()
	e := makeEntry("todo-def", "todo", "default scan", "open", "", "")
	writeEntries(t, dir, today(), []entry.Entry{e})

	todos, _, err := ScanOpenItems(dir, 0) // 0 should default to 30
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("expected 1 todo with default scanDays, got %d", len(todos))
	}
}

// TestScanOpenItems_originalEntryPreserved verifies that Entry field contains
// the original entry content (not the mutation's content).
func TestScanOpenItems_originalEntryPreserved(t *testing.T) {
	dir := t.TempDir()

	orig := makeEntry("todo-orig", "todo", "original content", "open", "", "")
	orig.Project = "myproject"

	mutation := makeEntry("todo-orig-v2", "todo", "updated content", "open", "todo-orig", "")
	mutation.Project = "otherproject"

	writeEntries(t, dir, today(), []entry.Entry{orig, mutation})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Entry.Content != "original content" {
		t.Errorf("Entry should have original content, got %q", todos[0].Entry.Content)
	}
	if todos[0].Entry.Project != "myproject" {
		t.Errorf("Entry should have original project, got %q", todos[0].Entry.Project)
	}
	if todos[0].LatestState.ID != "todo-orig-v2" {
		t.Errorf("LatestState should be mutation, got ID %q", todos[0].LatestState.ID)
	}
}

// TestScanOpenItems_emptyFile verifies no error and empty results for an empty
// JSONL file.
func TestScanOpenItems_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := entryFilePath(dir, today())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	todos, blockers, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if len(todos) != 0 || len(blockers) != 0 {
		t.Errorf("expected empty results, got todos=%d blockers=%d", len(todos), len(blockers))
	}
}

// TestScanOpenItems_chainAcrossDays verifies supersedes chains that span
// multiple JSONL files (original on day 1, mutation on day 2) are resolved.
func TestScanOpenItems_chainAcrossDays(t *testing.T) {
	dir := t.TempDir()

	day1 := today().AddDate(0, 0, -5)
	day2 := today()

	orig := makeEntryWithTime("todo-cross", "todo", "cross-day task", "open", "", "", day1)
	mutation := makeEntryWithTime("todo-cross-v2", "todo", "cross-day task", "abandoned", "todo-cross", "", day2)

	writeEntries(t, dir, day1, []entry.Entry{orig})
	writeEntries(t, dir, day2, []entry.Entry{mutation})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos (cross-day abandoned), got %d", len(todos))
	}
}

// TestScanOpenItems_reopenedCrossDay verifies a cross-day reopen scenario.
func TestScanOpenItems_reopenedCrossDay(t *testing.T) {
	dir := t.TempDir()

	day1 := today().AddDate(0, 0, -5)
	day2 := today().AddDate(0, 0, -3)
	day3 := today()

	orig := makeEntryWithTime("todo-reopen", "todo", "cross reopen", "open", "", "", day1)
	closed := makeEntryWithTime("todo-reopen-v2", "todo", "cross reopen", "abandoned", "todo-reopen", "", day2)
	reopened := makeEntryWithTime("todo-reopen-v3", "todo", "cross reopen", "open", "todo-reopen-v2", "", day3)

	writeEntries(t, dir, day1, []entry.Entry{orig})
	writeEntries(t, dir, day2, []entry.Entry{closed})
	writeEntries(t, dir, day3, []entry.Entry{reopened})

	todos, _, err := ScanOpenItems(dir, 30)
	if err != nil {
		t.Fatalf("ScanOpenItems: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 open todo (cross-day reopen), got %d", len(todos))
	}
	if todos[0].LatestState.ID != "todo-reopen-v3" {
		t.Errorf("expected LatestState to be reopen entry, got %q", todos[0].LatestState.ID)
	}
	if todos[0].AgeDays != 5 {
		t.Errorf("expected AgeDays=5, got %d", todos[0].AgeDays)
	}
}

// ---------------------------------------------------------------------------
// entryFilePath helper test
// ---------------------------------------------------------------------------

func TestEntryFilePath(t *testing.T) {
	cases := []struct {
		date     time.Time
		wantSufx string
	}{
		{time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC), fmt.Sprintf("entries%c2026%c05%c2026-05-16.jsonl", filepath.Separator, filepath.Separator, filepath.Separator)},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), fmt.Sprintf("entries%c2026%c01%c2026-01-01.jsonl", filepath.Separator, filepath.Separator, filepath.Separator)},
	}
	for _, tc := range cases {
		got := entryFilePath("/data", tc.date)
		want := filepath.Join("/data", tc.wantSufx)
		if got != want {
			t.Errorf("entryFilePath(%v) = %q, want %q", tc.date, got, want)
		}
	}
}
