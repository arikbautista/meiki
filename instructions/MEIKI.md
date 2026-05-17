# MEIKI.md — AI Session Instructions

## Lifecycle Triggers

- **Session start:** Run `meiki brief --json`. If non-empty, present conversationally. If empty, proceed silently.
- **Session end / "wrap up" / "done for the day":** Run `meiki review`. Do not surface output unless user asks.
- **"Good morning" / "let's get to work":** Re-run `meiki brief --json` (self-debounces).

## In-Session Capture

Capture work as it happens. One command per event:

- `meiki capture achievement "Deployed auth service to prod"` — completed work
- `meiki capture achievement --closes <id> "Fixed the login bug"` — complete a tracked todo
- `meiki capture learning "Redis SCAN cursor must be treated as opaque"` — non-obvious insight
- `meiki capture blocker "CI runner out of disk space"` — blocker encountered
- `meiki capture todo --priority tomorrow "Write migration rollback test"` — new todo (priority: tomorrow, this-week, someday)
- `meiki capture idea "Could replace polling with SSE for dashboard"` — half-formed thought

## Mutations

- Todo done: prefer `--closes <id>` on an achievement capture.
- Todo dropped: `meiki abandon <id> "Superseded by new approach"`
- Blocker resolved: `meiki resolve <id> "Upgraded runner disk to 100GB"`
- User correction: `meiki reopen <id>` — only run when user explicitly requests.

## Stale Item Triage

When `meiki brief --json` includes "needs_triage" items, ask the user about each before starting other work. Options: re-prioritize, set new due date, or abandon.

## Anti-Patterns — Do NOT

- Capture AI reasoning, tool calls, file navigation, or trivial actions.
- Duplicate entries — run `meiki today` to check before capturing.
- Capture what did not actually happen (plans, intentions, speculation).
- Call `meiki reopen` on your own — only when user explicitly asks.

## Edge Cases

- If `meiki` is not on PATH, silently skip all meiki calls. Mention once per session if desired.
- Empty stdout from any `meiki` command means no-op. Do not mention meiki to the user.
- Never fabricate entry IDs. Use only IDs returned by previous meiki commands.
