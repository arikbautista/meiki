# Story: Abandon Command

**Epic:** CLI Commands
**Ref:** Design spec §6.2, §5.5

## Summary

Implement `meiki abandon <id> [reason]` — marks an open todo as abandoned by writing a new mutation entry.

## Scope

### Usage

```
meiki abandon <id> ["reason for abandoning"]
```

### Behavior

- Validates `<id>` references an existing open todo
- Writes a new entry: `{type: "todo", status: "abandoned", supersedes: <id>, content: "<reason>"}`
- If no reason provided, content defaults to "abandoned"
- Prints the new mutation entry's id to stdout

### Errors

- Unknown id → exit 1, "entry not found"
- Id references a non-todo → exit 1, "can only abandon todos"
- Id references an already-closed todo → exit 1, "todo is already <status>"

## Acceptance Criteria

- [ ] Abandoning an open todo produces a mutation entry with `supersedes` pointing to the original
- [ ] `meiki open` no longer lists the abandoned todo
- [ ] Reason is captured in the mutation entry's content
- [ ] Rejects non-todo entries, already-closed todos, and unknown ids with appropriate errors
