# Story: Review Command

**Epic:** CLI Commands
**Ref:** Design spec §6.1, §5.2, §7.3

## Summary

Implement `meiki review [--silent]` — generates today's daily review as a markdown file. Idempotent and safe to call repeatedly. Called by the AI at session end and by the Stop hook.

## Scope

### Usage

```
meiki review [--silent]
```

### Review markdown generation

Reads today's JSONL entries and produces a structured markdown file at `<data_dir>/reviews/YYYY/MM/YYYY-MM-DD.md`:

```markdown
# Daily Review — 2026-05-15

## Achievements
- Shipped the capture command (meiki)
- Fixed JSONL atomic write bug (meiki)

## Learnings
- O_APPEND is atomic under PIPE_BUF on POSIX

## Blockers
- CI pipeline broken on arm64 [resolved: switched to cross-compile]

## Ideas
- Weekly rollup command for v2

## Open Items
- [tomorrow] Ping Sam re: API contract (meiki) — 2 days overdue
- [this-week] Update README with install instructions (meiki)
```

- Template-based, no LLM — deterministic output from entries
- Groups by type; includes open items (from scanner) at the bottom
- Blockers show their resolution status if resolved during the day
- Idempotent: regenerating overwrites the same file with current data

### --silent flag

Suppresses stdout output. The review file is still written. Used by the Stop hook so the user doesn't see output when a session ends.

### State update

After generating, updates `last_review_ts` in `state.json`.

### Edge cases

- No entries today: writes a minimal review ("No entries recorded today.") and still updates the timestamp
- Creates year/month directories under `reviews/` if they don't exist

## Acceptance Criteria

- [ ] Generates a well-formatted markdown review at the correct path
- [ ] Review includes all entry types grouped with open items at the bottom
- [ ] Idempotent — running twice produces the same file
- [ ] `--silent` suppresses stdout but still writes the file
- [ ] Updates `last_review_ts` in state.json
- [ ] Creates intermediate directories as needed
- [ ] Handles days with no entries gracefully
