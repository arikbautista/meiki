# Story: Open-Item Scanner

**Epic:** Foundation
**Ref:** Design spec §5.3, §5.5

## Summary

Implement the scanner that derives open todos and blockers from the append-only JSONL log by walking entries backwards and resolving supersedes chains.

## Scope

### Core logic

Scan JSONL files in `entries/` walking back from today up to `open_scan_days` (default 30, from config). For each entry with a status field:

1. Collect all entries with `type: todo` or `type: blocker`
2. Build a supersedes graph: if entry B has `supersedes: A`, then B is the latest state for that logical item
3. For achievements with `closes: <id>`, treat the referenced todo as done
4. An item is "open" if its latest state-bearing entry has `status: open`

### Interface

```go
type OpenItem struct {
    Entry       Entry    // the original entry (for content, project, tags)
    LatestState Entry    // the most recent mutation (for current status)
    AgeDays     int      // days since original capture
}

func ScanOpenItems(dataDir string, scanDays int) (todos []OpenItem, blockers []OpenItem, err error)
```

### Edge cases

- An item with multiple mutations: follow the full chain, use the terminal state
- An item closed then reopened: the reopen is the latest state → item is open
- Entries from today and historical entries are both included
- Missing JSONL files for some days in the range: skip silently

## Acceptance Criteria

- [ ] Returns open todos and open blockers separately
- [ ] Correctly resolves `supersedes` chains (A → B → C, only C's status matters)
- [ ] `closes` on an achievement marks the referenced todo as done
- [ ] Reopened items appear as open
- [ ] Respects `scanDays` parameter (doesn't look further back)
- [ ] Handles missing files, empty files, and days with no entries
- [ ] Tests cover: simple open, closed via achievement, abandoned, resolved blocker, reopen after close, multi-step chain
