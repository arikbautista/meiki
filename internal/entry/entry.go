// Package entry provides the entry model, ULID generation, and JSONL I/O.
package entry

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/oklog/ulid/v2"
)

// ValidTypes is the closed set of allowed entry types.
var ValidTypes = map[string]bool{
	"achievement": true,
	"learning":    true,
	"blocker":     true,
	"todo":        true,
	"idea":        true,
}

// ValidPriorities is the closed set of allowed priority values.
var ValidPriorities = map[string]bool{
	"tomorrow":  true,
	"this-week": true,
	"someday":   true,
}

// Entry represents a single work-memory entry.
type Entry struct {
	ID          string   `json:"id"`
	Timestamp   string   `json:"ts"`
	Type        string   `json:"type"`
	Content     string   `json:"content"`
	Project     string   `json:"project,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Source      string   `json:"source,omitempty"`
	ExternalRef string   `json:"external_ref,omitempty"`
	Supersedes  string   `json:"supersedes,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Due         string   `json:"due,omitempty"`
	Status      string   `json:"status,omitempty"`
	Closes      string   `json:"closes,omitempty"`

	// Extras holds any JSON fields not recognised by the struct so they are
	// preserved on read and re-emitted on write.
	Extras map[string]json.RawMessage `json:"-"`
}

// MarshalJSON implements custom JSON marshalling that re-emits unknown fields.
func (e Entry) MarshalJSON() ([]byte, error) {
	// Build a map from the known fields.
	m := make(map[string]interface{})
	m["id"] = e.ID
	m["ts"] = e.Timestamp
	m["type"] = e.Type
	m["content"] = e.Content
	if e.Project != "" {
		m["project"] = e.Project
	}
	if len(e.Tags) > 0 {
		m["tags"] = e.Tags
	}
	if e.Source != "" {
		m["source"] = e.Source
	}
	if e.ExternalRef != "" {
		m["external_ref"] = e.ExternalRef
	}
	if e.Supersedes != "" {
		m["supersedes"] = e.Supersedes
	}
	if e.Priority != "" {
		m["priority"] = e.Priority
	}
	if e.Due != "" {
		m["due"] = e.Due
	}
	if e.Status != "" {
		m["status"] = e.Status
	}
	if e.Closes != "" {
		m["closes"] = e.Closes
	}
	// Merge extras (unknown fields), extras do NOT override known fields.
	for k, v := range e.Extras {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshalling that preserves unknown fields.
func (e *Entry) UnmarshalJSON(data []byte) error {
	// First, unmarshal known fields using a type alias to avoid recursion.
	type plain struct {
		ID          string   `json:"id"`
		Timestamp   string   `json:"ts"`
		Type        string   `json:"type"`
		Content     string   `json:"content"`
		Project     string   `json:"project,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Source      string   `json:"source,omitempty"`
		ExternalRef string   `json:"external_ref,omitempty"`
		Supersedes  string   `json:"supersedes,omitempty"`
		Priority    string   `json:"priority,omitempty"`
		Due         string   `json:"due,omitempty"`
		Status      string   `json:"status,omitempty"`
		Closes      string   `json:"closes,omitempty"`
	}
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	e.ID = p.ID
	e.Timestamp = p.Timestamp
	e.Type = p.Type
	e.Content = p.Content
	e.Project = p.Project
	e.Tags = p.Tags
	e.Source = p.Source
	e.ExternalRef = p.ExternalRef
	e.Supersedes = p.Supersedes
	e.Priority = p.Priority
	e.Due = p.Due
	e.Status = p.Status
	e.Closes = p.Closes

	// Now capture all fields into a raw map to find extras.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	knownKeys := map[string]bool{
		"id": true, "ts": true, "type": true, "content": true,
		"project": true, "tags": true, "source": true, "external_ref": true,
		"supersedes": true, "priority": true, "due": true, "status": true,
		"closes": true,
	}
	extras := make(map[string]json.RawMessage)
	for k, v := range raw {
		if !knownKeys[k] {
			extras[k] = v
		}
	}
	if len(extras) > 0 {
		e.Extras = extras
	}
	return nil
}

// monotonic entropy source, safe for concurrent use.
var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0) //nolint:gosec
)

// NewID returns a new ULID string using monotonic entropy.
func NewID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), entropy).String()
}

// NewEntry constructs a minimal valid Entry with a fresh ULID and UTC timestamp.
// The caller must set at least Type and Content before the entry is valid.
func NewEntry(entryType, content string) (*Entry, error) {
	e := &Entry{
		ID:        NewID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Type:      entryType,
		Content:   content,
	}
	if err := Validate(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Validate checks that an entry meets all structural requirements.
func Validate(e *Entry) error {
	if e.Content == "" {
		return fmt.Errorf("entry content is required")
	}
	if !ValidTypes[e.Type] {
		return fmt.Errorf("invalid entry type %q: must be one of achievement, learning, blocker, todo, idea", e.Type)
	}
	if e.Priority != "" && !ValidPriorities[e.Priority] {
		return fmt.Errorf("invalid priority %q: must be one of tomorrow, this-week, someday", e.Priority)
	}
	if e.Priority != "" && e.Type != "todo" {
		return fmt.Errorf("priority is only valid on todo entries, got type %q", e.Type)
	}
	if e.Due != "" && e.Type != "todo" {
		return fmt.Errorf("due is only valid on todo entries, got type %q", e.Type)
	}
	if e.Closes != "" && e.Type != "achievement" {
		return fmt.Errorf("closes is only valid on achievement entries, got type %q", e.Type)
	}
	return nil
}

// EntryFilePath returns the path to the JSONL file for the given date.
// Date must be a time.Time value; only the year, month, and day are used.
func EntryFilePath(date time.Time) string {
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	return filepath.Join(config.DataDir(), "entries", y, m, d+".jsonl")
}

// AppendEntry marshals e to JSON and atomically appends it to the daily JSONL file.
// It creates intermediate year/month directories as needed.
// Returns the entry's ID.
func AppendEntry(e *Entry) (string, error) {
	path := EntryFilePath(time.Now().UTC())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create entry directory: %w", err)
	}

	line, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("marshal entry: %w", err)
	}
	line = append(line, '\n')

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return "", fmt.Errorf("open entry file: %w", err)
	}
	defer f.Close()

	// Single Write call keeps the write within PIPE_BUF for atomicity.
	if _, err := f.Write(line); err != nil {
		return "", fmt.Errorf("write entry: %w", err)
	}
	return e.ID, nil
}

// AppendEntryToPath is like AppendEntry but writes to an explicit file path.
// Intermediate directories are created as needed.
// Used by tests to isolate file I/O from the real data directory.
func AppendEntryToPath(e *Entry, path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create entry directory: %w", err)
	}

	line, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("marshal entry: %w", err)
	}
	line = append(line, '\n')

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return "", fmt.Errorf("open entry file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return "", fmt.Errorf("write entry: %w", err)
	}
	return e.ID, nil
}

// ReadEntries reads all entries from a single day's JSONL file.
// It returns an empty slice (no error) if the file does not exist.
// Blank lines and malformed lines are skipped with a log warning.
func ReadEntries(date time.Time) ([]Entry, error) {
	path := EntryFilePath(date)
	return readEntriesFromPath(path)
}

// ReadEntriesRange reads entries across an inclusive date range [from, to].
// Dates are compared at day granularity.
func ReadEntriesRange(from, to time.Time) ([]Entry, error) {
	var result []Entry
	// Normalise to start of each day.
	cur := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	for !cur.After(end) {
		entries, err := ReadEntries(cur)
		if err != nil {
			return result, err
		}
		result = append(result, entries...)
		cur = cur.AddDate(0, 0, 1)
	}
	return result, nil
}

// ReadEntriesFromPath reads all entries from an explicit JSONL file path.
// It returns an empty slice (no error) if the file does not exist.
// Blank lines and malformed lines are skipped with a log warning.
// Used by the scanner package and tests that need path-based access.
func ReadEntriesFromPath(path string) ([]Entry, error) {
	return readEntriesFromPath(path)
}

// readEntriesFromPath reads entries from an explicit file path.
// Used internally and by tests.
func readEntriesFromPath(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("open entry file: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			log.Printf("entry: skipping malformed line %d in %s: %v", lineNum, path, err)
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("scan entry file: %w", err)
	}
	return entries, nil
}
