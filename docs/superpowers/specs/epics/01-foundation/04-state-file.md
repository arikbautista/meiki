# Story: State File Management

**Epic:** Foundation
**Ref:** Design spec §5.2

## Summary

Implement read/write for `state.json`, which tracks timestamps for debouncing `brief` and `review` commands.

## Scope

### State model

```go
type State struct {
    LastBriefTS  string `json:"last_brief_ts"`   // ISO-8601 UTC
    LastReviewTS string `json:"last_review_ts"`   // ISO-8601 UTC
}
```

- File location: `<data_dir>/state.json`
- Both fields are full timestamps (not dates) so `brief` can detect "new entries since last brief" within the same day

### Operations

- `LoadState()` — reads state.json; returns zero-value State if file doesn't exist
- `SaveState(state)` — writes state.json atomically (write to temp file, rename)
- `UpdateBriefTS(now)` — loads, updates `last_brief_ts`, saves
- `UpdateReviewTS(now)` — loads, updates `last_review_ts`, saves

## Acceptance Criteria

- [ ] `LoadState()` returns zero-value when file doesn't exist (no error)
- [ ] `SaveState()` writes valid JSON readable by `LoadState()`
- [ ] State file updates are atomic (temp file + rename, not in-place write)
- [ ] `UpdateBriefTS()` and `UpdateReviewTS()` update only their respective field
