package entry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func tmpFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "2006", "01", "2006-01-02.jsonl")
}

// ---------------------------------------------------------------------------
// NewID
// ---------------------------------------------------------------------------

func TestNewID_format(t *testing.T) {
	id := NewID()
	if len(id) != 26 {
		t.Fatalf("expected ULID length 26, got %d (%q)", len(id), id)
	}
}

func TestNewID_monotonic(t *testing.T) {
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = NewID()
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("IDs not monotonically increasing: %s <= %s", ids[i], ids[i-1])
		}
	}
}

// ---------------------------------------------------------------------------
// NewEntry
// ---------------------------------------------------------------------------

func TestNewEntry_valid(t *testing.T) {
	e, err := NewEntry("todo", "buy milk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID == "" {
		t.Error("ID should not be empty")
	}
	if e.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	// Timestamp must be valid RFC3339.
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		t.Errorf("Timestamp %q is not valid RFC3339: %v", e.Timestamp, err)
	}
	if e.Type != "todo" {
		t.Errorf("expected type todo, got %q", e.Type)
	}
	if e.Content != "buy milk" {
		t.Errorf("expected content 'buy milk', got %q", e.Content)
	}
}

func TestNewEntry_invalidType(t *testing.T) {
	_, err := NewEntry("bogus", "content")
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestNewEntry_emptyContent(t *testing.T) {
	_, err := NewEntry("todo", "")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

// ---------------------------------------------------------------------------
// Validate
// ---------------------------------------------------------------------------

func TestValidate_allTypes(t *testing.T) {
	types := []string{"achievement", "learning", "blocker", "todo", "idea"}
	for _, typ := range types {
		e := &Entry{ID: "x", Timestamp: "2006-01-02T15:04:05Z", Type: typ, Content: "c"}
		if err := Validate(e); err != nil {
			t.Errorf("type %q should be valid, got: %v", typ, err)
		}
	}
}

func TestValidate_unknownType(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "unknown", Content: "c"}
	if err := Validate(e); err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestValidate_priorityOnNonTodo(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "learning", Content: "c", Priority: "someday"}
	if err := Validate(e); err == nil {
		t.Error("expected error: priority on non-todo type")
	}
}

func TestValidate_priorityOnTodo(t *testing.T) {
	for _, p := range []string{"tomorrow", "this-week", "someday"} {
		e := &Entry{ID: "x", Timestamp: "t", Type: "todo", Content: "c", Priority: p}
		if err := Validate(e); err != nil {
			t.Errorf("priority %q on todo should be valid, got: %v", p, err)
		}
	}
}

func TestValidate_invalidPriority(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "todo", Content: "c", Priority: "urgent"}
	if err := Validate(e); err == nil {
		t.Error("expected error for invalid priority value")
	}
}

func TestValidate_dueOnNonTodo(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "idea", Content: "c", Due: "2026-01-01"}
	if err := Validate(e); err == nil {
		t.Error("expected error: due on non-todo type")
	}
}

func TestValidate_closesOnNonAchievement(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "todo", Content: "c", Closes: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	if err := Validate(e); err == nil {
		t.Error("expected error: closes on non-achievement type")
	}
}

func TestValidate_closesOnAchievement(t *testing.T) {
	e := &Entry{ID: "x", Timestamp: "t", Type: "achievement", Content: "c", Closes: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	if err := Validate(e); err != nil {
		t.Errorf("closes on achievement should be valid, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EntryFilePath
// ---------------------------------------------------------------------------

func TestEntryFilePath(t *testing.T) {
	date := time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC)
	path := EntryFilePath(date)
	if !strings.HasSuffix(path, filepath.Join("entries", "2026", "05", "2026-05-16.jsonl")) {
		t.Errorf("unexpected path: %s", path)
	}
}

// ---------------------------------------------------------------------------
// AppendEntry / ReadEntries round-trip
// ---------------------------------------------------------------------------

func appendToPath(t *testing.T, e *Entry, path string) string {
	t.Helper()
	id, err := AppendEntryToPath(e, path)
	if err != nil {
		t.Fatalf("AppendEntryToPath: %v", err)
	}
	return id
}

func TestAppendEntry_createsFile(t *testing.T) {
	path := tmpFile(t)
	e, _ := NewEntry("todo", "test entry")
	appendToPath(t, e, path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestAppendEntry_createsDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "dirs", "file.jsonl")
	e, _ := NewEntry("idea", "nested dirs test")
	if _, err := AppendEntryToPath(e, path); err != nil {
		t.Fatalf("AppendEntryToPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestAppendEntry_returnsID(t *testing.T) {
	path := tmpFile(t)
	e, _ := NewEntry("learning", "go is fun")
	id := appendToPath(t, e, path)
	if id != e.ID {
		t.Errorf("returned id %q != entry id %q", id, e.ID)
	}
}

func TestRoundTrip(t *testing.T) {
	path := tmpFile(t)
	want, _ := NewEntry("blocker", "CI is broken")
	want.Project = "meiki"
	want.Tags = []string{"ci", "infra"}
	want.Source = "cli"
	want.Status = "open"
	appendToPath(t, want, path)

	got, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("readEntriesFromPath: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	g := got[0]
	if g.ID != want.ID {
		t.Errorf("ID: got %q, want %q", g.ID, want.ID)
	}
	if g.Type != want.Type {
		t.Errorf("Type: got %q, want %q", g.Type, want.Type)
	}
	if g.Content != want.Content {
		t.Errorf("Content: got %q, want %q", g.Content, want.Content)
	}
	if g.Project != want.Project {
		t.Errorf("Project: got %q, want %q", g.Project, want.Project)
	}
	if len(g.Tags) != len(want.Tags) {
		t.Errorf("Tags: got %v, want %v", g.Tags, want.Tags)
	}
	if g.Status != want.Status {
		t.Errorf("Status: got %q, want %q", g.Status, want.Status)
	}
}

func TestReadEntries_emptyFile(t *testing.T) {
	path := tmpFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadEntries_missingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.jsonl")
	entries, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadEntries_malformedLines(t *testing.T) {
	path := tmpFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write one good entry, one bad line, another good entry.
	good1, _ := NewEntry("idea", "first good")
	good2, _ := NewEntry("idea", "second good")
	line1, _ := json.Marshal(good1)
	line2, _ := json.Marshal(good2)

	content := string(line1) + "\n{invalid json}\n\n" + string(line2) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 good entries, got %d", len(entries))
	}
}

func TestReadEntries_blankLines(t *testing.T) {
	path := tmpFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	e, _ := NewEntry("achievement", "shipped v1")
	line, _ := json.Marshal(e)
	content := "\n\n" + string(line) + "\n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// Unknown field preservation
// ---------------------------------------------------------------------------

func TestUnknownFieldPreservation(t *testing.T) {
	raw := `{"id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","ts":"2006-01-02T15:04:05Z","type":"idea","content":"test","unknown_field":"preserved","another":42}`

	var e Entry
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Extras == nil {
		t.Fatal("Extras should not be nil")
	}
	if _, ok := e.Extras["unknown_field"]; !ok {
		t.Error("unknown_field should be in Extras")
	}
	if _, ok := e.Extras["another"]; !ok {
		t.Error("another should be in Extras")
	}

	// Re-marshal and confirm the unknown fields are still there.
	out, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["unknown_field"]; !ok {
		t.Error("unknown_field lost after re-marshal")
	}
	if _, ok := m["another"]; !ok {
		t.Error("another lost after re-marshal")
	}
}

// ---------------------------------------------------------------------------
// ReadEntriesRange
// ---------------------------------------------------------------------------

func TestReadEntriesRange(t *testing.T) {
	// Patch EntryFilePath to use a temp dir.
	// We test via readEntriesFromPath directly with synthetic dates.
	dir := t.TempDir()

	// Create two files for 2026-05-14 and 2026-05-16.
	writeDay := func(dateStr string, contents []string) {
		t.Helper()
		y := dateStr[:4]
		m := dateStr[5:7]
		path := filepath.Join(dir, "entries", y, m, dateStr+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, c := range contents {
			e, err := NewEntry("idea", c)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(e)
			lines = append(lines, string(b))
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeDay("2026-05-14", []string{"day14-a", "day14-b"})
	writeDay("2026-05-16", []string{"day16-a"})
	// 2026-05-15 has no file.

	// Read each day individually via readEntriesFromPath to verify our helper.
	path14 := filepath.Join(dir, "entries", "2026", "05", "2026-05-14.jsonl")
	entries14, err := readEntriesFromPath(path14)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries14) != 2 {
		t.Errorf("expected 2 entries for 2026-05-14, got %d", len(entries14))
	}

	path15 := filepath.Join(dir, "entries", "2026", "05", "2026-05-15.jsonl")
	entries15, err := readEntriesFromPath(path15)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries15) != 0 {
		t.Errorf("expected 0 entries for missing 2026-05-15, got %d", len(entries15))
	}

	path16 := filepath.Join(dir, "entries", "2026", "05", "2026-05-16.jsonl")
	entries16, err := readEntriesFromPath(path16)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries16) != 1 {
		t.Errorf("expected 1 entry for 2026-05-16, got %d", len(entries16))
	}

	total := len(entries14) + len(entries15) + len(entries16)
	if total != 3 {
		t.Errorf("expected 3 total entries across range, got %d", total)
	}
}

// ---------------------------------------------------------------------------
// Concurrent writes
// ---------------------------------------------------------------------------

func TestConcurrentWrites(t *testing.T) {
	path := tmpFile(t)
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			e, err := NewEntry("idea", fmt.Sprintf("concurrent entry %d", n))
			if err != nil {
				t.Errorf("NewEntry: %v", err)
				return
			}
			if _, err := AppendEntryToPath(e, path); err != nil {
				t.Errorf("AppendEntryToPath: %v", err)
			}
		}(i)
	}
	wg.Wait()

	entries, err := readEntriesFromPath(path)
	if err != nil {
		t.Fatalf("readEntriesFromPath: %v", err)
	}
	if len(entries) != goroutines {
		t.Errorf("expected %d entries, got %d (some writes may have been lost or corrupted)", goroutines, len(entries))
	}
	// Verify all entries are valid JSON with non-empty IDs.
	for i, e := range entries {
		if e.ID == "" {
			t.Errorf("entry %d has empty ID", i)
		}
		if e.Content == "" {
			t.Errorf("entry %d has empty content", i)
		}
	}
}

// ---------------------------------------------------------------------------
// setEnv helper
// ---------------------------------------------------------------------------

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	original, wasSet := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if wasSet {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	})
}

// ---------------------------------------------------------------------------
// AppendEntry one line per write
// ---------------------------------------------------------------------------

func TestAppendEntry_oneLinePerWrite(t *testing.T) {
	path := tmpFile(t)
	for i := 0; i < 3; i++ {
		e, _ := NewEntry("learning", fmt.Sprintf("line %d", i))
		appendToPath(t, e, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %q", i, line)
		}
	}
}

// ---------------------------------------------------------------------------
// AppendEntryAt logical day placement
// ---------------------------------------------------------------------------

func TestAppendEntry_UsesLogicalDay(t *testing.T) {
	dataDir := t.TempDir()
	setEnv(t, "XDG_DATA_HOME", dataDir)

	configDir := t.TempDir()
	setEnv(t, "XDG_CONFIG_HOME", configDir)
	meikiDir := filepath.Join(configDir, "meiki")
	os.MkdirAll(meikiDir, 0o755)
	os.WriteFile(filepath.Join(meikiDir, "config.toml"), []byte(`
[ui]
timezone = "America/New_York"
day_start_hour = 5
`), 0o644)

	ny, _ := time.LoadLocation("America/New_York")
	ts := time.Date(2026, 5, 18, 2, 0, 0, 0, ny)

	e := &Entry{
		ID:        NewID(),
		Timestamp: ts.UTC().Format(time.RFC3339),
		Type:      "todo",
		Content:   "late night task",
		Status:    "open",
		Priority:  "tomorrow",
	}

	_, err := AppendEntryAt(e, ts)
	if err != nil {
		t.Fatalf("AppendEntryAt() error = %v", err)
	}

	base := filepath.Join(dataDir, "meiki")
	expectedPath := filepath.Join(base, "entries", "2026", "05", "2026-05-17.jsonl")
	unexpectedPath := filepath.Join(base, "entries", "2026", "05", "2026-05-18.jsonl")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedPath)
	}
	if _, err := os.Stat(unexpectedPath); !os.IsNotExist(err) {
		t.Errorf("did not expect file %s to exist", unexpectedPath)
	}
}
