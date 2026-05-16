# Story: Today Command

**Epic:** CLI Commands
**Ref:** Design spec §6.3

## Summary

Implement `meiki today [--json]` — displays today's entries grouped by type. Used for mid-session "what have I logged so far?" checks.

## Scope

### Usage

```
meiki today [--json]
```

### Human-readable output

Groups entries by type in order: achievements, learnings, blockers, todos, ideas.

```
Achievements (2):
  [01HXY9...] "Shipped the capture command" (meiki)
  [01HXY9...] "Fixed JSONL atomic write bug" (meiki)

Todos (1):
  [01HXY9...] tomorrow "Write tests for scanner" (meiki)

Ideas (1):
  [01HXY9...] "Weekly rollup command for v2" (meiki)
```

- Mutation entries (abandon, resolve, reopen) are included with a label, e.g., "[abandoned] reason"
- Empty state: "Nothing logged today." and exit 0

### JSON output

Array of today's entries, unmodified from JSONL (no computed fields needed).

## Acceptance Criteria

- [ ] Shows all entries from today's JSONL file, grouped by type
- [ ] Type group order: achievements, learnings, blockers, todos, ideas
- [ ] Mutation entries are visually distinguished
- [ ] `--json` returns the raw entries array
- [ ] Empty state handled gracefully
