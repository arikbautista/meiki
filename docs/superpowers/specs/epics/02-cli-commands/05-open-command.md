# Story: Open Command

**Epic:** CLI Commands
**Ref:** Design spec §6.3

## Summary

Implement `meiki open [--json]` — displays all open todos and blockers, grouped by type.

## Scope

### Usage

```
meiki open [--json]
```

### Human-readable output

```
Open Todos (3):
  [01HXY9...] tomorrow  "Ping Sam re: API contract" (meiki, 2 days overdue)
  [01HXY9...] this-week "Update README with install instructions" (meiki)
  [01HXY9...] someday   "Look into caching layer" (meiki)

Open Blockers (1):
  [01HXY9...] "CI pipeline broken on arm64" (meiki)
```

- Todos sorted by priority (tomorrow > this-week > someday), then by capture date
- Each item shows: truncated id, priority (todos), content, project, overdue age if applicable
- Empty state: "No open items." and exit 0

### JSON output

Array of open items with full entry data plus computed fields (`age_days`, `overdue_days`, `triage` classification).

### Implementation

Uses `ScanOpenItems()` and `ClassifyItem()` from Foundation stories.

## Acceptance Criteria

- [ ] Lists open todos and blockers grouped by type
- [ ] Todos are sorted by priority then capture date
- [ ] Overdue items show their age
- [ ] `--json` produces structured output with computed fields
- [ ] Empty state handled gracefully
- [ ] Exit code 0 in all cases (read-only command)
