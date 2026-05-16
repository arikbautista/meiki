# Story: Brief Command

**Epic:** CLI Commands
**Ref:** Design spec §6.1, §5.3, §5.4, §7.3

## Summary

Implement `meiki brief [--json]` — the morning briefing that presents yesterday's review summary, open items ranked by priority, and stale items needing triage. Debounced to avoid redundant output.

## Scope

### Usage

```
meiki brief [--json]
```

### Briefing content

1. **Yesterday's review summary** — reads the most recent review markdown (may not be yesterday if the user skipped days) and includes a condensed version
2. **Open todos** — ranked by priority (tomorrow > this-week > someday), limited to `brief_max_open_todos` (default 20 from config)
3. **Open blockers** — listed after todos
4. **Needs triage section** — items 3+ days overdue (per `stale_triage_days` config), separated from the main list
5. **Overdue flags** — items 1-3 days overdue are shown in the main list with their age (e.g., "2 days overdue")

### Debouncing

- On invocation, check `last_brief_ts` in state.json
- If a brief was already produced today AND no new entries exist since `last_brief_ts`: output nothing (empty stdout), exit 0
- "New entries since last brief" = any entries in today's JSONL with timestamp > `last_brief_ts`
- After producing output, update `last_brief_ts`

### First-run behavior

On a fresh install with no history (no entries, no reviews): output a one-line welcome message, e.g., "Welcome to meiki. Entries you capture during AI sessions will appear in your next briefing."

### JSON output

Structured object with sections: `review_summary`, `open_todos`, `open_blockers`, `needs_triage`, `overdue_items`.

## Acceptance Criteria

- [ ] Produces a briefing with review summary, open items, and blockers
- [ ] Open todos sorted by priority then capture date
- [ ] Items 1-3 days overdue are flagged with their age
- [ ] Items 3+ days overdue appear in a "Needs triage" section
- [ ] Respects `brief_max_open_todos` limit from config
- [ ] Debounces: same-day repeat with no new entries → empty stdout, exit 0
- [ ] Debounce resets when new entries are captured after the last brief
- [ ] Fresh install with no data → welcome message
- [ ] `--json` produces structured output
- [ ] Updates `last_brief_ts` after producing output
