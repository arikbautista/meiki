# Story: Reopen Command

**Epic:** CLI Commands
**Ref:** Design spec §6.2, §5.5

## Summary

Implement `meiki reopen <id>` — undoes a close, abandon, or resolve by writing a new mutation entry that restores open status.

## Scope

### Usage

```
meiki reopen <id>
```

### Behavior

- Validates `<id>` references an existing non-open todo or blocker
- Writes a new entry: `{type: <original_type>, status: "open", supersedes: <id>, content: "reopened"}`
- The `supersedes` field references the most recent state-bearing entry for that logical item (which may be a mutation entry, not the original)
- Prints the new mutation entry's id to stdout

### Errors

- Unknown id → exit 1, "entry not found"
- Id references an item that is already open → exit 1, "already open"
- Id references a type without status (learning, idea) → exit 1, "cannot reopen <type>"

## Acceptance Criteria

- [ ] Reopening a closed todo makes it appear in `meiki open` again
- [ ] Reopening a resolved blocker makes it appear in `meiki open` again
- [ ] Reopening an abandoned todo makes it appear in `meiki open` again
- [ ] The mutation entry preserves the original entry's type
- [ ] Rejects already-open items and types without status
