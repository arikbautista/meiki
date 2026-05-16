package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/entry"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runRecent executes newRecentCmd() with the given arguments using the provided
// dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runRecent(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newRecentCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// writeRecentEntry writes a single entry to the JSONL file for the given date.
func writeRecentEntry(t *testing.T, dataDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(dataDir, "meiki", "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeRecentEntry: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

func TestRecent_emptyState(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "No entries in the last 7 days." {
		t.Errorf("expected empty message, got %q", stdout)
	}
}

func TestRecent_emptyState_customDays(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runRecent(t, dataDir, "--days", "14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "No entries in the last 14 days." {
		t.Errorf("expected empty message with 14 days, got %q", stdout)
	}
}

func TestRecent_emptyState_json(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runRecent(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var items []entry.Entry
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

// ---------------------------------------------------------------------------
// Default: last 7 days, all types
// ---------------------------------------------------------------------------

func TestRecent_defaultShowsLast7Days(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Entry from 6 days ago (within default window).
	sixDaysAgo := now.AddDate(0, 0, -6)
	e1 := makeEntry("ACHID00000000000000000001", "achievement", "within window", "meiki", sixDaysAgo)
	writeRecentEntry(t, dataDir, sixDaysAgo, e1)

	// Entry from 8 days ago (outside default window).
	eightDaysAgo := now.AddDate(0, 0, -8)
	e2 := makeEntry("ACHID00000000000000000002", "achievement", "outside window", "meiki", eightDaysAgo)
	writeRecentEntry(t, dataDir, eightDaysAgo, e2)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "within window") {
		t.Errorf("expected entry within window in output:\n%s", stdout)
	}
	if strings.Contains(stdout, "outside window") {
		t.Errorf("expected entry outside window to be excluded:\n%s", stdout)
	}
}

func TestRecent_daysFlag(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Entry from 10 days ago — within a 14-day window, outside 7-day default.
	tenDaysAgo := now.AddDate(0, 0, -10)
	e := makeEntry("ACHID00000000000000000001", "achievement", "ten days ago", "meiki", tenDaysAgo)
	writeRecentEntry(t, dataDir, tenDaysAgo, e)

	// Default (7 days): should not appear.
	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, "ten days ago") {
		t.Errorf("entry outside default window should not appear:\n%s", stdout)
	}

	// With --days 14: should appear.
	stdout, err = runRecent(t, dataDir, "--days", "14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "ten days ago") {
		t.Errorf("entry within 14-day window should appear:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Type filter
// ---------------------------------------------------------------------------

func TestRecent_typeFilter(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e1 := makeEntry("ACHID00000000000000000001", "achievement", "shipped it", "meiki", now)
	e2 := makeEntry("LRNID00000000000000000001", "learning", "learned something", "meiki", now)
	writeRecentEntry(t, dataDir, now, e1)
	writeRecentEntry(t, dataDir, now, e2)

	stdout, err := runRecent(t, dataDir, "--type", "achievement")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "shipped it") {
		t.Errorf("expected achievement entry in output:\n%s", stdout)
	}
	if strings.Contains(stdout, "learned something") {
		t.Errorf("expected learning entry to be filtered out:\n%s", stdout)
	}
}

func TestRecent_typeFilter_allValidTypes(t *testing.T) {
	validTypes := []string{"achievement", "learning", "blocker", "todo", "idea"}
	dataDir := t.TempDir()

	for _, typeName := range validTypes {
		_, err := runRecent(t, dataDir, "--type", typeName)
		if err != nil {
			t.Errorf("--type %q should be valid, got error: %v", typeName, err)
		}
	}
}

func TestRecent_typeFilter_invalidType(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runRecent(t, dataDir, "--type", "invalid-type")
	if err == nil {
		t.Error("expected error for invalid type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid type") {
		t.Errorf("expected 'invalid type' in error, got: %v", err)
	}
}

func TestRecent_typeFilter_emptyResult(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Only achievements, but filter by learning.
	e := makeEntry("ACHID00000000000000000001", "achievement", "shipped it", "meiki", now)
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir, "--type", "learning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "No entries in the last") {
		t.Errorf("expected empty state message, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Human-readable output: grouping and ordering
// ---------------------------------------------------------------------------

func TestRecent_groupedByDateMostRecentFirst(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	e1 := makeEntry("ACHID00000000000000000001", "achievement", "today entry", "meiki", today)
	e2 := makeEntry("ACHID00000000000000000002", "achievement", "yesterday entry", "meiki", yesterday)
	writeRecentEntry(t, dataDir, today, e1)
	writeRecentEntry(t, dataDir, yesterday, e2)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todayDate := today.Format("2006-01-02")
	yesterdayDate := yesterday.Format("2006-01-02")

	if !strings.Contains(stdout, todayDate) {
		t.Errorf("expected today's date %q in output:\n%s", todayDate, stdout)
	}
	if !strings.Contains(stdout, yesterdayDate) {
		t.Errorf("expected yesterday's date %q in output:\n%s", yesterdayDate, stdout)
	}

	todayPos := strings.Index(stdout, todayDate)
	yesterdayPos := strings.Index(stdout, yesterdayDate)
	if todayPos > yesterdayPos {
		t.Errorf("expected today's date before yesterday's date (most recent first):\n%s", stdout)
	}
}

func TestRecent_typeOrderWithinDate(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Write entries in reverse order to verify canonical type ordering.
	writeRecentEntry(t, dataDir, now, makeEntry("IDEAID0000000000000000001", "idea", "idea content", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("TODOID0000000000000000001", "todo", "todo content", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("BLKID00000000000000000001", "blocker", "blocker content", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("LRNID00000000000000000001", "learning", "learning content", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "achievement content", "p", now))

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	achPos := strings.Index(stdout, "Achievements")
	lrnPos := strings.Index(stdout, "Learnings")
	blkPos := strings.Index(stdout, "Blockers")
	todPos := strings.Index(stdout, "Todos")
	ideaPos := strings.Index(stdout, "Ideas")

	if achPos < 0 || lrnPos < 0 || blkPos < 0 || todPos < 0 || ideaPos < 0 {
		t.Fatalf("not all group headers found:\n%s", stdout)
	}
	if !(achPos < lrnPos && lrnPos < blkPos && blkPos < todPos && todPos < ideaPos) {
		t.Errorf("expected type order achievements < learnings < blockers < todos < ideas:\n%s", stdout)
	}
}

func TestRecent_emptyDatesOmitted(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Entry only on today; yesterday should not appear even though it's in range.
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	e := makeEntry("ACHID00000000000000000001", "achievement", "today only", "meiki", today)
	writeRecentEntry(t, dataDir, today, e)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	if strings.Contains(stdout, yesterday) {
		t.Errorf("yesterday date %q should not appear when empty:\n%s", yesterday, stdout)
	}
}

func TestRecent_dateHeaderFormat(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACHID00000000000000000001", "achievement", "check date header", "meiki", now)
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Date header should be "YYYY-MM-DD:" format.
	dateStr := now.UTC().Format("2006-01-02")
	if !strings.Contains(stdout, dateStr+":") {
		t.Errorf("expected date header %q in output:\n%s", dateStr+":", stdout)
	}
}

// ---------------------------------------------------------------------------
// Human-readable output: entry formatting
// ---------------------------------------------------------------------------

func TestRecent_showsTruncatedID(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACHID00000000000000000001", "achievement", "check id", "meiki", now)
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "ACHID000") {
		t.Errorf("expected truncated ID prefix in output:\n%s", stdout)
	}
}

func TestRecent_todoWithPriority(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("TODOID0000000000000000001", "todo", "do this tomorrow", "meiki", now)
	e.Priority = "tomorrow"
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "tomorrow") {
		t.Errorf("expected priority 'tomorrow' in output:\n%s", stdout)
	}
}

func TestRecent_mutationEntry(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	mut := makeMutation("MUTID00000000000000000001", "todo", "abandoned", "TODOID0000000000000000001", "not needed", "meiki", now)
	writeRecentEntry(t, dataDir, now, mut)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "[abandoned]") {
		t.Errorf("expected '[abandoned]' label in output:\n%s", stdout)
	}
}

func TestRecent_missingProjectShowsUnknown(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACHID00000000000000000001", "achievement", "no project", "", now)
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "unknown") {
		t.Errorf("expected 'unknown' project fallback in output:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// JSON output
// ---------------------------------------------------------------------------

func TestRecent_json_returnsAllEntries(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	writeRecentEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "ach", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("LRNID00000000000000000001", "learning", "learn", "p", now))

	stdout, err := runRecent(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []entry.Entry
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestRecent_json_sortedByTimestampDescending(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-1 * time.Hour)

	e1 := makeEntry("ACHID00000000000000000001", "achievement", "earlier entry", "p", earlier)
	e2 := makeEntry("ACHID00000000000000000002", "achievement", "later entry", "p", later)
	writeRecentEntry(t, dataDir, now, e1)
	writeRecentEntry(t, dataDir, now, e2)

	stdout, err := runRecent(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []entry.Entry
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First item should be the later one (descending order).
	if items[0].Content != "later entry" {
		t.Errorf("expected 'later entry' first in JSON output (descending), got %q", items[0].Content)
	}
	if items[1].Content != "earlier entry" {
		t.Errorf("expected 'earlier entry' second in JSON output (descending), got %q", items[1].Content)
	}
}

func TestRecent_json_withTypeFilter(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	writeRecentEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "ach", "p", now))
	writeRecentEntry(t, dataDir, now, makeEntry("LRNID00000000000000000001", "learning", "learn", "p", now))

	stdout, err := runRecent(t, dataDir, "--json", "--type", "achievement")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []entry.Entry
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after type filter, got %d", len(items))
	}
	if items[0].Type != "achievement" {
		t.Errorf("expected type achievement, got %q", items[0].Type)
	}
}

func TestRecent_json_entryFields(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACHID00000000000000000001", "achievement", "shipped it", "myapp", now)
	writeRecentEntry(t, dataDir, now, e)

	stdout, err := runRecent(t, dataDir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []entry.Entry
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, stdout)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ID != "ACHID00000000000000000001" {
		t.Errorf("ID: want ACHID00000000000000000001, got %q", item.ID)
	}
	if item.Type != "achievement" {
		t.Errorf("Type: want achievement, got %q", item.Type)
	}
	if item.Content != "shipped it" {
		t.Errorf("Content: want 'shipped it', got %q", item.Content)
	}
	if item.Project != "myapp" {
		t.Errorf("Project: want myapp, got %q", item.Project)
	}
}

// ---------------------------------------------------------------------------
// Exit code is always 0
// ---------------------------------------------------------------------------

func TestRecent_exitCodeZeroOnEmpty(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runRecent(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) on empty state, got %v", err)
	}
}

func TestRecent_exitCodeZeroWithEntries(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("ACHID00000000000000000001", "achievement", "something", "p", now)
	writeRecentEntry(t, dataDir, now, e)

	_, err := runRecent(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) with entries, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// dateKey helper
// ---------------------------------------------------------------------------

func TestDateKey(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"2026-05-15T10:30:00Z", "2026-05-15"},
		{"2026-05-15T23:59:59Z", "2026-05-15"},
		{"2026-01-01T00:00:00Z", "2026-01-01"},
		// Fallback for malformed: first 10 chars.
		{"2026-05-15Tgarbage", "2026-05-15"},
	}

	for _, tc := range cases {
		got := dateKey(tc.input)
		if got != tc.want {
			t.Errorf("dateKey(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
