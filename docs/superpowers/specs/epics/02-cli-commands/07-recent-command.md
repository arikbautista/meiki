# Story: Recent Command

**Epic:** CLI Commands
**Ref:** Design spec §6.3

## Summary

Implement `meiki recent [--days N] [--type T] [--json]` — filtered tail of entries across multiple days.

## Scope

### Usage

```
meiki recent [--days N] [--type T] [--json]

Defaults:
  --days 7
  --type (all types)
```

### Human-readable output

Entries grouped by date (most recent first), then by type within each date.

```
2026-05-15:
  Achievements (2):
    [01HXY9...] "Shipped capture command" (meiki)
    [01HXY9...] "Fixed atomic write bug" (meiki)
  Todos (1):
    [01HXY9...] tomorrow "Write scanner tests" (meiki)

2026-05-14:
  Learnings (1):
    [01HXY9...] "O_APPEND is atomic under PIPE_BUF" (meiki)
```

- When `--type` is specified, only show entries of that type
- Empty state: "No entries in the last N days." and exit 0

### JSON output

Array of entries sorted by timestamp descending.

## Acceptance Criteria

- [ ] Defaults to last 7 days, all types
- [ ] `--days` controls the lookback window
- [ ] `--type` filters to a single entry type
- [ ] `--type` validates against the closed type set
- [ ] `--json` returns filtered entries as an array
- [ ] Entries grouped by date then type in human-readable mode
- [ ] Days with no matching entries are omitted (not shown as empty)
