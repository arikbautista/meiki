# Epic 2: CLI Commands

**Goal:** Implement all v1 CLI commands — capture, mutations, queries, review, and brief.

**Depends on:** Epic 1 (Foundation)

**Done when:** All commands from design spec §6 work end-to-end against real JSONL files, produce correct human-readable and JSON output, and handle edge cases documented in the spec.

## Stories

| # | Story | Summary |
|---|-------|---------|
| 1 | [Capture Command](01-capture-command.md) | `meiki capture` with all type/option combinations |
| 2 | [Abandon Command](02-abandon-command.md) | `meiki abandon <id> [reason]` |
| 3 | [Resolve Command](03-resolve-command.md) | `meiki resolve <id> [how]` |
| 4 | [Reopen Command](04-reopen-command.md) | `meiki reopen <id>` |
| 5 | [Open Command](05-open-command.md) | `meiki open [--json]` — open todos + blockers |
| 6 | [Today Command](06-today-command.md) | `meiki today [--json]` — today's entries |
| 7 | [Recent Command](07-recent-command.md) | `meiki recent [--days N] [--type T] [--json]` |
| 8 | [Review Command](08-review-command.md) | `meiki review [--silent]` — generate daily review markdown |
| 9 | [Brief Command](09-brief-command.md) | `meiki brief [--json]` — morning briefing with debounce and stale triage |

## Implementation Notes

- All commands follow exit code conventions: 0 success, 1 user error, 2 internal error
- Capture commands print entry id on stdout, diagnostics on stderr
- `--json` flag available on query and lifecycle commands
- Empty stdout = no-op (debounced calls)
