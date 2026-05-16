# Story: Priority Decay & Stale Detection

**Epic:** Foundation
**Ref:** Design spec §5.4

## Summary

Implement overdue calculation and stale-item classification so that `brief` can flag aging items and present a "Needs triage" section.

## Scope

### Overdue rules

Given an open todo and today's date:

| Condition | Overdue when |
|-----------|-------------|
| `due` is set | today > `due` date |
| `priority: "tomorrow"` (no due) | today > capture date + 1 day |
| `priority: "this-week"` (no due) | today > end of the ISO week the todo was captured in |
| `priority: "someday"` (no due) | never overdue |

Explicit `due` takes precedence over priority-based calculation.

### Classification

```go
type ItemTriage int
const (
    TriageNormal   ItemTriage = iota  // not overdue
    TriageOverdue                      // 1-3 days overdue — flag with age
    TriageStale                        // 3+ days overdue — "Needs triage" section
)

func ClassifyItem(item OpenItem, today time.Time, staleDays int) (ItemTriage, int)
// Returns classification and days overdue (0 if not overdue)
```

- `staleDays` comes from config (`stale_triage_days`, default 3)
- Days overdue is calculated from the overdue threshold, not from capture date

### Integration with OpenItem

Extend or wrap `ScanOpenItems` output to include triage classification for each item.

## Acceptance Criteria

- [ ] `priority: "tomorrow"` item captured yesterday is overdue by 1 day today
- [ ] `priority: "tomorrow"` item captured today is not overdue
- [ ] `priority: "this-week"` item captured Monday is overdue on the following Monday
- [ ] `priority: "someday"` items are never overdue
- [ ] `due: "2026-05-10"` item is overdue on 2026-05-11 regardless of priority
- [ ] Items 1-2 days overdue are classified `TriageOverdue`
- [ ] Items 3+ days overdue are classified `TriageStale` (with default staleDays=3)
- [ ] `staleDays` config is respected
- [ ] Tests cover all priority levels, due date override, and boundary conditions
