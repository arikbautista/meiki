package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runOpen executes newOpenCmd() with the given arguments using the provided
// dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runOpen(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newOpenCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// writeTestEntry writes a single entry to the JSONL file for the given date in
// XDG_DATA_HOME/meiki/entries/YYYY/MM/YYYY-MM-DD.jsonl.
func writeTestEntry(t *testing.T, dataDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(dataDir, "meiki", "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeTestEntry: %v", err)
	}
}

// makeOpenTodo returns an open todo entry with the given fields.
func makeOpenTodo(id, content, project, priority string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:        id,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Type:      "todo",
		Content:   content,
		Project:   project,
		Priority:  priority,
		Status:    "open",
	}
}

// makeOpenBlocker returns an open blocker entry.
func makeOpenBlocker(id, content, project string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:        id,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Type:      "blocker",
		Content:   content,
		Project:   project,
		Status:    "open",
	}
}

// todayMidnight returns today at midnight UTC.
func todayMidnightUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

func TestOpen_emptyState(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "No open items." {
		t.Errorf("expected %q, got %q", "No open items.", stdout)
	}
}

func TestOpen_emptyState_json(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runOpen(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var items []openJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

// ---------------------------------------------------------------------------
// Human-readable output
// ---------------------------------------------------------------------------

func TestOpen_singleTodo(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenTodo("TODOID00000000000000000001", "do the thing", "myapp", "this-week", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Open Todos (1):") {
		t.Errorf("expected header 'Open Todos (1):' in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "do the thing") {
		t.Errorf("expected content in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "myapp") {
		t.Errorf("expected project in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "TODOID00") {
		t.Errorf("expected truncated ID prefix in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "...") {
		t.Errorf("expected truncated ID suffix '...' in output:\n%s", stdout)
	}
}

func TestOpen_singleBlocker(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenBlocker("BLKID00000000000000000001", "CI pipeline broken", "myapp", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Open Blockers (1):") {
		t.Errorf("expected 'Open Blockers (1):' in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "CI pipeline broken") {
		t.Errorf("expected blocker content in output:\n%s", stdout)
	}
}

func TestOpen_noBlockerSectionWhenEmpty(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenTodo("TODOID00000000000000000001", "only a todo", "proj", "someday", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "Open Blockers") {
		t.Errorf("should not show Blockers section when empty:\n%s", stdout)
	}
}

func TestOpen_noTodoSectionWhenEmpty(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenBlocker("BLKID00000000000000000001", "only a blocker", "proj", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "Open Todos") {
		t.Errorf("should not show Todos section when empty:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Priority sort order
// ---------------------------------------------------------------------------

func TestOpen_todosSortedByPriority(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()

	// Write in reverse priority order to verify sorting.
	someday := makeOpenTodo("TODOID00000000000000000003", "someday task", "proj", "someday", now)
	thisWeek := makeOpenTodo("TODOID00000000000000000002", "this-week task", "proj", "this-week", now)
	tomorrow := makeOpenTodo("TODOID00000000000000000001", "tomorrow task", "proj", "tomorrow", now)

	writeTestEntry(t, dataDir, now, someday)
	writeTestEntry(t, dataDir, now, thisWeek)
	writeTestEntry(t, dataDir, now, tomorrow)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tomorrowPos := strings.Index(stdout, "tomorrow task")
	thisWeekPos := strings.Index(stdout, "this-week task")
	somedayPos := strings.Index(stdout, "someday task")

	if tomorrowPos < 0 || thisWeekPos < 0 || somedayPos < 0 {
		t.Fatalf("not all tasks found in output:\n%s", stdout)
	}
	if !(tomorrowPos < thisWeekPos && thisWeekPos < somedayPos) {
		t.Errorf("expected priority order tomorrow < this-week < someday, positions: %d %d %d\n%s",
			tomorrowPos, thisWeekPos, somedayPos, stdout)
	}
}

// TestOpen_samePrioritySortedByTimestamp verifies that within the same priority
// level, items are sorted by capture timestamp (earlier first).
func TestOpen_samePrioritySortedByTimestamp(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-1 * time.Hour)

	e1 := makeOpenTodo("TODOID00000000000000000001", "later task", "proj", "this-week", later)
	e2 := makeOpenTodo("TODOID00000000000000000002", "earlier task", "proj", "this-week", earlier)

	writeTestEntry(t, dataDir, now, e1)
	writeTestEntry(t, dataDir, now, e2)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	earlierPos := strings.Index(stdout, "earlier task")
	laterPos := strings.Index(stdout, "later task")

	if earlierPos < 0 || laterPos < 0 {
		t.Fatalf("tasks not found in output:\n%s", stdout)
	}
	if earlierPos > laterPos {
		t.Errorf("expected earlier task before later task in output:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Overdue display
// ---------------------------------------------------------------------------

func TestOpen_overdueShowsAge(t *testing.T) {
	dataDir := t.TempDir()
	// A "tomorrow" priority todo captured 3 days ago is overdue.
	threeDaysAgo := todayMidnightUTC().AddDate(0, 0, -3)
	e := makeOpenTodo("TODOID00000000000000000001", "overdue task", "proj", "tomorrow", threeDaysAgo)
	writeTestEntry(t, dataDir, threeDaysAgo, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "overdue") {
		t.Errorf("expected 'overdue' in output for overdue item:\n%s", stdout)
	}
	if !strings.Contains(stdout, "days overdue") {
		t.Errorf("expected 'days overdue' in output:\n%s", stdout)
	}
}

func TestOpen_nonOverdueNoAgeShown(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	// "someday" is never overdue.
	e := makeOpenTodo("TODOID00000000000000000001", "someday task", "proj", "someday", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "overdue") {
		t.Errorf("expected no 'overdue' for someday task:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// JSON output
// ---------------------------------------------------------------------------

func TestOpen_json_fields(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenTodo("TODOID00000000000000000001", "json task", "myapp", "this-week", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []openJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ID != "TODOID00000000000000000001" {
		t.Errorf("ID: want TODOID00000000000000000001, got %q", item.ID)
	}
	if item.Type != "todo" {
		t.Errorf("Type: want todo, got %q", item.Type)
	}
	if item.Content != "json task" {
		t.Errorf("Content: want 'json task', got %q", item.Content)
	}
	if item.Project != "myapp" {
		t.Errorf("Project: want myapp, got %q", item.Project)
	}
	if item.Priority != "this-week" {
		t.Errorf("Priority: want this-week, got %q", item.Priority)
	}
	if item.Triage == "" {
		t.Error("Triage should not be empty")
	}
}

func TestOpen_json_blocker_type(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenBlocker("BLKID00000000000000000001", "json blocker", "proj", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []openJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "blocker" {
		t.Errorf("expected type blocker, got %q", items[0].Type)
	}
}

func TestOpen_json_overdueDays(t *testing.T) {
	dataDir := t.TempDir()
	// "tomorrow" captured 3 days ago → 3 days overdue.
	threeDaysAgo := todayMidnightUTC().AddDate(0, 0, -3)
	e := makeOpenTodo("TODOID00000000000000000001", "overdue json task", "proj", "tomorrow", threeDaysAgo)
	writeTestEntry(t, dataDir, threeDaysAgo, e)

	stdout, err := runOpen(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []openJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].OverdueDays != 3 {
		t.Errorf("OverdueDays: want 3, got %d", items[0].OverdueDays)
	}
	if items[0].Triage != "overdue" && items[0].Triage != "stale" {
		t.Errorf("Triage: want overdue or stale, got %q", items[0].Triage)
	}
	if items[0].AgeDays != 3 {
		t.Errorf("AgeDays: want 3, got %d", items[0].AgeDays)
	}
}

func TestOpen_json_triageNormal(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	// "someday" is never overdue → triage normal.
	e := makeOpenTodo("TODOID00000000000000000001", "someday task", "proj", "someday", now)
	writeTestEntry(t, dataDir, now, e)

	stdout, err := runOpen(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []openJSONItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Triage != "normal" {
		t.Errorf("Triage: want normal, got %q", items[0].Triage)
	}
}

// ---------------------------------------------------------------------------
// truncateID helper
// ---------------------------------------------------------------------------

func TestTruncateID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"01HXY9ZZZZZZZZZZZZZZZZZZZZ", "01HXY9ZZ..."},
		{"12345678ABCDEFGH", "12345678..."},
		{"short", "short"},
		{"12345678", "12345678"},
		{"123456789", "12345678..."},
	}

	for _, tc := range cases {
		got := truncateID(tc.input)
		if got != tc.want {
			t.Errorf("truncateID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// triageName helper
// ---------------------------------------------------------------------------

func TestTriageName(t *testing.T) {
	cases := []struct {
		input scanner.ItemTriage
		want  string
	}{
		{scanner.TriageNormal, "normal"},
		{scanner.TriageOverdue, "overdue"},
		{scanner.TriageStale, "stale"},
	}

	for _, tc := range cases {
		got := triageName(tc.input)
		if got != tc.want {
			t.Errorf("triageName(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Exit code is always 0
// ---------------------------------------------------------------------------

func TestOpen_exitCodeZeroOnEmpty(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runOpen(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) on empty state, got %v", err)
	}
}

func TestOpen_exitCodeZeroWithItems(t *testing.T) {
	dataDir := t.TempDir()
	now := todayMidnightUTC()
	e := makeOpenTodo("TODOID00000000000000000001", "something", "proj", "someday", now)
	writeTestEntry(t, dataDir, now, e)

	_, err := runOpen(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) with items, got %v", err)
	}
}
