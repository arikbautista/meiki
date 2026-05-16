# Story: Resolve Command

**Epic:** CLI Commands
**Ref:** Design spec §6.2, §5.5

## Summary

Implement `meiki resolve <id> [how]` — marks an open blocker as resolved by writing a new mutation entry.

## Scope

### Usage

```
meiki resolve <id> ["how it was resolved"]
```

### Behavior

- Validates `<id>` references an existing open blocker
- Writes a new entry: `{type: "blocker", status: "resolved", supersedes: <id>, content: "<how>"}`
- If no description provided, content defaults to "resolved"
- Prints the new mutation entry's id to stdout

### Errors

- Unknown id → exit 1, "entry not found"
- Id references a non-blocker → exit 1, "can only resolve blockers"
- Id references an already-resolved blocker → exit 1, "blocker is already resolved"

## Acceptance Criteria

- [ ] Resolving an open blocker produces a mutation entry with `supersedes`
- [ ] `meiki open` no longer lists the resolved blocker
- [ ] Resolution description is captured in content
- [ ] Rejects non-blocker entries, already-resolved blockers, and unknown ids
