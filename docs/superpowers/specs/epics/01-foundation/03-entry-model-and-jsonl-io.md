# Story: Entry Model & JSONL I/O

**Epic:** Foundation
**Ref:** Design spec §5.1, §5.2, §5.5

## Summary

Define the entry struct matching the spec's schema, implement ULID generation, and build JSONL read/write operations with atomic append semantics.

## Scope

### Entry model

```go
type Entry struct {
    ID          string   `json:"id"`
    Timestamp   string   `json:"ts"`
    Type        string   `json:"type"`        // achievement|learning|blocker|todo|idea
    Content     string   `json:"content"`
    Project     string   `json:"project,omitempty"`
    Tags        []string `json:"tags,omitempty"`
    Source      string   `json:"source,omitempty"`     // cli|hook|manual
    ExternalRef string   `json:"external_ref,omitempty"`
    Supersedes  string   `json:"supersedes,omitempty"`
    Priority    string   `json:"priority,omitempty"`   // tomorrow|this-week|someday (todo only)
    Due         string   `json:"due,omitempty"`         // ISO date (todo only)
    Status      string   `json:"status,omitempty"`      // open|done|abandoned (todo), open|resolved (blocker)
    Closes      string   `json:"closes,omitempty"`      // todo id (achievement only)
}
```

- Valid types: `achievement`, `learning`, `blocker`, `todo`, `idea`
- Valid priorities: `tomorrow`, `this-week`, `someday`
- Validation: type is required and must be in the closed set; content is required; type-specific field validation (e.g., priority only on todos)
- Unknown JSON fields are preserved on read (permissive parser per spec)

### ULID generation

- Use `oklog/ulid` or equivalent
- Monotonic within a millisecond for ordering guarantees

### JSONL file paths

- `EntryFilePath(date) → <data_dir>/entries/YYYY/MM/YYYY-MM-DD.jsonl`
- Create intermediate directories (`YYYY/MM/`) on write if they don't exist

### Write

- `AppendEntry(entry)` — marshals to JSON, appends one line to the daily JSONL file
- File opened with `O_WRONLY|O_APPEND|O_CREATE`, mode 0644
- Each write is a single `Write()` call (fits within PIPE_BUF for atomicity)
- Returns the entry's ID

### Read

- `ReadEntries(date) → []Entry` — reads all entries from a single day's JSONL file
- `ReadEntriesRange(from, to) → []Entry` — reads entries across a date range
- Skip blank lines; log and skip malformed lines (don't crash on partial writes)

## Acceptance Criteria

- [ ] `NewEntry()` produces a valid entry with a ULID id and UTC timestamp
- [ ] Type validation rejects unknown types
- [ ] `AppendEntry()` writes exactly one JSON line to the correct daily file
- [ ] `AppendEntry()` creates intermediate year/month directories
- [ ] File is opened with `O_APPEND` for concurrent write safety
- [ ] `ReadEntries()` round-trips entries through write→read
- [ ] `ReadEntries()` handles empty files, missing files (returns empty slice, no error), and malformed lines gracefully
- [ ] Unknown JSON fields in existing entries are preserved on read (not stripped)
- [ ] Tests cover concurrent writes (multiple goroutines appending)
