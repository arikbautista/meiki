package review

import (
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

// defaultConfig returns a Config with sensible defaults for tests.
func defaultConfig() config.Config {
	return config.Config{
		UI: config.UIConfig{
			OpenScanDays:    30,
			StaleTriageDays: 3,
		},
	}
}

// writeEntry writes a single entry to the JSONL file for the given date in
// dataDir/entries/YYYY/MM/YYYY-MM-DD.jsonl.
func writeEntry(t *testing.T, dataDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(dataDir, "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeEntry: %v", err)
	}
}

// makeEntry returns a minimal entry for tests.
func makeEntry(id, entryType, content, project string, ts time.Time) entry.Entry {
	return entry.Entry{
		ID:        id,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Type:      entryType,
		Content:   content,
		Project:   project,
	}
}

// ---------------------------------------------------------------------------
// No entries today
// ---------------------------------------------------------------------------

func TestGenerateReview_noEntries(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "# Daily Review") {
		t.Errorf("expected '# Daily Review' header, got:\n%s", md)
	}
	if !strings.Contains(md, "No entries recorded today.") {
		t.Errorf("expected 'No entries recorded today.' message, got:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Date in header
// ---------------------------------------------------------------------------

func TestGenerateReview_dateInHeader(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	date := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)

	md, err := GenerateReview(dataDir, date, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "# Daily Review — 2026-05-15") {
		t.Errorf("expected date header '# Daily Review — 2026-05-15', got:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Achievements section
// ---------------------------------------------------------------------------

func TestGenerateReview_achievements(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	e := makeEntry("ACH001", "achievement", "Shipped the capture command", "meiki", now)
	writeEntry(t, dataDir, now, e)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "## Achievements") {
		t.Errorf("expected '## Achievements' section, got:\n%s", md)
	}
	if !strings.Contains(md, "Shipped the capture command") {
		t.Errorf("expected achievement content in review:\n%s", md)
	}
	if !strings.Contains(md, "meiki") {
		t.Errorf("expected project name in review:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Learnings section
// ---------------------------------------------------------------------------

func TestGenerateReview_learnings(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	e := makeEntry("LRN001", "learning", "O_APPEND is atomic under PIPE_BUF on POSIX", "", now)
	writeEntry(t, dataDir, now, e)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "## Learnings") {
		t.Errorf("expected '## Learnings' section, got:\n%s", md)
	}
	if !strings.Contains(md, "O_APPEND is atomic under PIPE_BUF on POSIX") {
		t.Errorf("expected learning content in review:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Blockers section — unresolved
// ---------------------------------------------------------------------------

func TestGenerateReview_blockers_open(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	e := makeEntry("BLK001", "blocker", "CI pipeline broken on arm64", "meiki", now)
	e.Status = "open"
	writeEntry(t, dataDir, now, e)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "## Blockers") {
		t.Errorf("expected '## Blockers' section, got:\n%s", md)
	}
	if !strings.Contains(md, "CI pipeline broken on arm64") {
		t.Errorf("expected blocker content in review:\n%s", md)
	}
	if strings.Contains(md, "[resolved:") {
		t.Errorf("should not show [resolved:] for open blocker:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Blockers section — resolved today
// ---------------------------------------------------------------------------

func TestGenerateReview_blockers_resolvedToday(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	// Original blocker.
	orig := makeEntry("BLK001", "blocker", "CI pipeline broken on arm64", "meiki", now)
	orig.Status = "open"
	writeEntry(t, dataDir, now, orig)

	// Resolution mutation entry written today.
	mut := entry.Entry{
		ID:         "MUT001",
		Timestamp:  now.UTC().Format(time.RFC3339),
		Type:       "blocker",
		Content:    "switched to cross-compile",
		Project:    "meiki",
		Status:     "resolved",
		Supersedes: "BLK001",
	}
	writeEntry(t, dataDir, now, mut)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "[resolved: switched to cross-compile]") {
		t.Errorf("expected '[resolved: switched to cross-compile]' in review:\n%s", md)
	}
	// The mutation entry itself should not appear as a separate bullet.
	lines := strings.Split(md, "\n")
	blockerLines := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "- CI pipeline broken") {
			blockerLines++
		}
	}
	if blockerLines != 1 {
		t.Errorf("expected exactly 1 blocker line, got %d:\n%s", blockerLines, md)
	}
}

// ---------------------------------------------------------------------------
// Ideas section
// ---------------------------------------------------------------------------

func TestGenerateReview_ideas(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	e := makeEntry("IDEA001", "idea", "Weekly rollup command for v2", "", now)
	writeEntry(t, dataDir, now, e)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "## Ideas") {
		t.Errorf("expected '## Ideas' section, got:\n%s", md)
	}
	if !strings.Contains(md, "Weekly rollup command for v2") {
		t.Errorf("expected idea content in review:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// Open Items section
// ---------------------------------------------------------------------------

func TestGenerateReview_openItems_todos(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	// Write a todo entry (open).
	todo := entry.Entry{
		ID:        "TODO001",
		Timestamp: now.UTC().Format(time.RFC3339),
		Type:      "todo",
		Content:   "Ping Sam re: API contract",
		Project:   "meiki",
		Priority:  "tomorrow",
		Status:    "open",
	}
	writeEntry(t, dataDir, now, todo)

	// Also write an achievement so today's log has entries.
	ach := makeEntry("ACH001", "achievement", "did something", "meiki", now)
	writeEntry(t, dataDir, now, ach)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "## Open Items") {
		t.Errorf("expected '## Open Items' section, got:\n%s", md)
	}
	if !strings.Contains(md, "Ping Sam re: API contract") {
		t.Errorf("expected todo content in Open Items:\n%s", md)
	}
	if !strings.Contains(md, "[tomorrow]") {
		t.Errorf("expected priority label [tomorrow] in Open Items:\n%s", md)
	}
}

func TestGenerateReview_openItems_overdue(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	// Write an overdue todo (captured 3 days ago, priority=tomorrow).
	threeDaysAgo := now.AddDate(0, 0, -3)
	todo := entry.Entry{
		ID:        "TODO002",
		Timestamp: threeDaysAgo.UTC().Format(time.RFC3339),
		Type:      "todo",
		Content:   "Fix overdue task",
		Project:   "myapp",
		Priority:  "tomorrow",
		Status:    "open",
	}
	writeEntry(t, dataDir, threeDaysAgo, todo)

	// Write today's entry so the review has content.
	ach := makeEntry("ACH001", "achievement", "something", "p", now)
	writeEntry(t, dataDir, now, ach)

	md, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "overdue") {
		t.Errorf("expected 'overdue' label for overdue todo:\n%s", md)
	}
}

// ---------------------------------------------------------------------------
// ReviewFilePath correctness
// ---------------------------------------------------------------------------

func TestReviewFilePath(t *testing.T) {
	dataDir := "/tmp/meiki-test"
	date := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	got := ReviewFilePath(dataDir, date)
	want := "/tmp/meiki-test/reviews/2026/05/2026-05-15.md"
	if got != want {
		t.Errorf("ReviewFilePath: want %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// Idempotency: same markdown on repeated calls
// ---------------------------------------------------------------------------

func TestGenerateReview_idempotent(t *testing.T) {
	dataDir := t.TempDir()
	cfg := defaultConfig()
	now := time.Now().UTC()

	e := makeEntry("ACH001", "achievement", "something great", "meiki", now)
	writeEntry(t, dataDir, now, e)

	md1, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	md2, err := GenerateReview(dataDir, now, cfg)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if md1 != md2 {
		t.Errorf("GenerateReview is not idempotent:\nfirst:\n%s\nsecond:\n%s", md1, md2)
	}
}

// ---------------------------------------------------------------------------
// buildResolutions helper
// ---------------------------------------------------------------------------

func TestBuildResolutions(t *testing.T) {
	now := time.Now().UTC()
	entries := []entry.Entry{
		{
			ID:         "MUT001",
			Timestamp:  now.Format(time.RFC3339),
			Type:       "blocker",
			Status:     "resolved",
			Supersedes: "BLK001",
			Content:    "switched to cross-compile",
		},
		{
			ID:        "BLK001",
			Timestamp: now.Format(time.RFC3339),
			Type:      "blocker",
			Status:    "open",
			Content:   "CI broken",
		},
	}

	resolutions := buildResolutions(entries)
	if got, ok := resolutions["BLK001"]; !ok {
		t.Error("expected BLK001 in resolutions map")
	} else if got != "switched to cross-compile" {
		t.Errorf("expected resolution 'switched to cross-compile', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// plural helper
// ---------------------------------------------------------------------------

func TestPlural(t *testing.T) {
	cases := []struct {
		word  string
		count int
		want  string
	}{
		{"day", 1, "day"},
		{"day", 2, "days"},
		{"day", 0, "days"},
		{"item", 5, "items"},
	}
	for _, tc := range cases {
		got := plural(tc.word, tc.count)
		if got != tc.want {
			t.Errorf("plural(%q, %d) = %q, want %q", tc.word, tc.count, got, tc.want)
		}
	}
}
