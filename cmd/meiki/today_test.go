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

// writeTestConfig writes a config.toml with UTC timezone and day_start_hour=0
// to the given cfgDir so tests use consistent time boundaries regardless of
// the machine's local timezone.
func writeTestConfig(t *testing.T, cfgDir string) {
	t.Helper()
	meikiCfgDir := filepath.Join(cfgDir, "meiki")
	if err := os.MkdirAll(meikiCfgDir, 0o755); err != nil {
		t.Fatalf("writeTestConfig: mkdir: %v", err)
	}
	cfgContent := "[ui]\ntimezone = \"UTC\"\nday_start_hour = 0\n"
	if err := os.WriteFile(filepath.Join(meikiCfgDir, "config.toml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("writeTestConfig: write: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runToday executes newTodayCmd() with the given arguments using the provided
// dataDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// A config with UTC timezone and day_start_hour=0 is written so tests use
// consistent time boundaries regardless of the machine's local timezone.
// Returns stdout output and any error.
func runToday(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	cfgDir := t.TempDir()
	writeTestConfig(t, cfgDir)
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("XDG_DATA_HOME", dataDir)

	cmd := newTodayCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// writeTodayEntry writes a single entry to today's JSONL file for the given
// date in XDG_DATA_HOME/meiki/entries/YYYY/MM/YYYY-MM-DD.jsonl.
func writeTodayEntry(t *testing.T, dataDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(dataDir, "meiki", "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeTodayEntry: %v", err)
	}
}

// makeEntry returns a minimal entry for use in tests.
func makeEntry(id, entryType, content, project string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:        id,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Type:      entryType,
		Content:   content,
		Project:   project,
	}
}

// makeMutation returns a mutation entry (abandon/resolve/reopen) for tests.
func makeMutation(id, entryType, status, supersedes, content, project string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:         id,
		Timestamp:  ts.UTC().Format(time.RFC3339),
		Type:       entryType,
		Status:     status,
		Supersedes: supersedes,
		Content:    content,
		Project:    project,
	}
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

func TestToday_emptyState(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "Nothing logged today." {
		t.Errorf("expected %q, got %q", "Nothing logged today.", stdout)
	}
}

func TestToday_emptyState_json(t *testing.T) {
	dataDir := t.TempDir()
	stdout, err := runToday(t, dataDir, "--json")
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
// Human-readable output — basic display
// ---------------------------------------------------------------------------

func TestToday_singleAchievement(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("ACHID00000000000000000001", "achievement", "Shipped the capture command", "meiki", now)
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Achievements (1):") {
		t.Errorf("expected 'Achievements (1):' in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Shipped the capture command") {
		t.Errorf("expected content in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "meiki") {
		t.Errorf("expected project in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ACHID000") {
		t.Errorf("expected truncated ID in output:\n%s", stdout)
	}
}

func TestToday_singleLearning(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("LRNID00000000000000000001", "learning", "Learned about ULID monotonicity", "meiki", now)
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Learnings (1):") {
		t.Errorf("expected 'Learnings (1):' in output:\n%s", stdout)
	}
}

func TestToday_singleBlocker(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("BLKID00000000000000000001", "blocker", "CI is broken", "meiki", now)
	e.Status = "open"
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Blockers (1):") {
		t.Errorf("expected 'Blockers (1):' in output:\n%s", stdout)
	}
}

func TestToday_singleTodo(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("TODOID0000000000000000001", "todo", "Write tests for scanner", "meiki", now)
	e.Priority = "tomorrow"
	e.Status = "open"
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Todos (1):") {
		t.Errorf("expected 'Todos (1):' in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "tomorrow") {
		t.Errorf("expected priority 'tomorrow' in todo line:\n%s", stdout)
	}
}

func TestToday_singleIdea(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("IDEAID0000000000000000001", "idea", "Weekly rollup command for v2", "meiki", now)
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Ideas (1):") {
		t.Errorf("expected 'Ideas (1):' in output:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Group ordering: achievements, learnings, blockers, todos, ideas
// ---------------------------------------------------------------------------

func TestToday_groupOrder(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Write entries in reverse order to verify output order is deterministic.
	writeTodayEntry(t, dataDir, now, makeEntry("IDEAID0000000000000000001", "idea", "idea content", "p", now))
	writeTodayEntry(t, dataDir, now, makeEntry("TODOID0000000000000000001", "todo", "todo content", "p", now))
	writeTodayEntry(t, dataDir, now, makeEntry("BLKID00000000000000000001", "blocker", "blocker content", "p", now))
	writeTodayEntry(t, dataDir, now, makeEntry("LRNID00000000000000000001", "learning", "learning content", "p", now))
	writeTodayEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "achievement content", "p", now))

	stdout, err := runToday(t, dataDir)
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
		t.Errorf("expected group order achievements < learnings < blockers < todos < ideas, positions: %d %d %d %d %d\n%s",
			achPos, lrnPos, blkPos, todPos, ideaPos, stdout)
	}
}

// ---------------------------------------------------------------------------
// Empty groups are not shown
// ---------------------------------------------------------------------------

func TestToday_onlyShowsNonEmptyGroups(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	writeTodayEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "shipped it", "p", now))

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "Learnings") {
		t.Errorf("should not show Learnings section when empty:\n%s", stdout)
	}
	if strings.Contains(stdout, "Blockers") {
		t.Errorf("should not show Blockers section when empty:\n%s", stdout)
	}
	if strings.Contains(stdout, "Todos") {
		t.Errorf("should not show Todos section when empty:\n%s", stdout)
	}
	if strings.Contains(stdout, "Ideas") {
		t.Errorf("should not show Ideas section when empty:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Mutation entries
// ---------------------------------------------------------------------------

func TestToday_abandonedMutation(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	orig := makeEntry("TODOID0000000000000000001", "todo", "do the thing", "meiki", now)
	orig.Status = "open"
	writeTodayEntry(t, dataDir, now, orig)

	mut := makeMutation("MUTID00000000000000000001", "todo", "abandoned", "TODOID0000000000000000001", "decided not to pursue", "meiki", now)
	writeTodayEntry(t, dataDir, now, mut)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "[abandoned]") {
		t.Errorf("expected '[abandoned]' label for mutation entry:\n%s", stdout)
	}
	if !strings.Contains(stdout, "decided not to pursue") {
		t.Errorf("expected mutation reason in output:\n%s", stdout)
	}
}

func TestToday_resolvedMutation(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	orig := makeEntry("TODOID0000000000000000001", "todo", "do the thing", "meiki", now)
	orig.Status = "open"
	writeTodayEntry(t, dataDir, now, orig)

	mut := makeMutation("MUTID00000000000000000001", "todo", "resolved", "TODOID0000000000000000001", "completed", "meiki", now)
	writeTodayEntry(t, dataDir, now, mut)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "[resolved]") {
		t.Errorf("expected '[resolved]' label for mutation entry:\n%s", stdout)
	}
}

func TestToday_reopenedMutation(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	orig := makeEntry("TODOID0000000000000000001", "todo", "do the thing", "meiki", now)
	orig.Status = "open"
	writeTodayEntry(t, dataDir, now, orig)

	// Reopen mutation: status="open", supersedes set.
	mut := makeMutation("MUTID00000000000000000001", "todo", "open", "TODOID0000000000000000001", "reopened", "meiki", now)
	writeTodayEntry(t, dataDir, now, mut)

	stdout, err := runToday(t, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "[reopened]") {
		t.Errorf("expected '[reopened]' label for reopen mutation:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Missing project falls back to "unknown"
// ---------------------------------------------------------------------------

func TestToday_missingProjectShowsUnknown(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("ACHID00000000000000000001", "achievement", "no project set", "", now)
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir)
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

func TestToday_json_returnsAllEntries(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	writeTodayEntry(t, dataDir, now, makeEntry("ACHID00000000000000000001", "achievement", "ach", "p", now))
	writeTodayEntry(t, dataDir, now, makeEntry("TODOID0000000000000000001", "todo", "todo", "p", now))

	stdout, err := runToday(t, dataDir, "--json")
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

func TestToday_json_entryFields(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACHID00000000000000000001", "achievement", "shipped it", "myapp", now)
	writeTodayEntry(t, dataDir, now, e)

	stdout, err := runToday(t, dataDir, "--json")
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

func TestToday_json_mutationEntry(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()

	mut := makeMutation("MUTID00000000000000000001", "todo", "abandoned", "TODOID0000000000000000001", "reason", "p", now)
	writeTodayEntry(t, dataDir, now, mut)

	stdout, err := runToday(t, dataDir, "--json")
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
	if items[0].Supersedes != "TODOID0000000000000000001" {
		t.Errorf("Supersedes: want TODOID0000000000000000001, got %q", items[0].Supersedes)
	}
	if items[0].Status != "abandoned" {
		t.Errorf("Status: want abandoned, got %q", items[0].Status)
	}
}

// ---------------------------------------------------------------------------
// mutationLabel helper
// ---------------------------------------------------------------------------

func TestMutationLabel(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"abandoned", "abandoned"},
		{"resolved", "resolved"},
		{"open", "reopened"},
		{"ABANDONED", "abandoned"},
		{"custom", "custom"},
		{"", "mutated"},
	}

	for _, tc := range cases {
		got := mutationLabel(tc.status)
		if got != tc.want {
			t.Errorf("mutationLabel(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Exit code is always 0
// ---------------------------------------------------------------------------

func TestToday_exitCodeZeroOnEmpty(t *testing.T) {
	dataDir := t.TempDir()
	_, err := runToday(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) on empty state, got %v", err)
	}
}

func TestToday_exitCodeZeroWithEntries(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC()
	e := makeEntry("ACHID00000000000000000001", "achievement", "something", "p", now)
	writeTodayEntry(t, dataDir, now, e)

	_, err := runToday(t, dataDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0) with entries, got %v", err)
	}
}
