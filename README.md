# meiki

Work memory for AI CLI sessions. Captures achievements, learnings, blockers, todos, and ideas during your sessions — then produces daily reviews and next-morning briefings.

## Install

```bash
# From source
go install github.com/arikbautista/meiki/cmd/meiki@latest

# Or build locally
make build
```

## Quick Start

```bash
meiki setup      # creates data dirs, prints integration snippets
meiki doctor     # verifies installation health
```

Paste the MEIKI.md instructions from `meiki setup` output into `~/.claude/CLAUDE.md` (or your AI tool's equivalent). The AI handles capture automatically during sessions.

## Commands

| Command | Description |
|---------|-------------|
| `meiki capture <type> "content"` | Log an entry (achievement, learning, blocker, todo, idea) |
| `meiki today` | Show today's entries |
| `meiki open` | List open todos and blockers |
| `meiki recent` | Filtered multi-day tail |
| `meiki review` | Generate daily review markdown |
| `meiki brief` | Morning briefing with open items and triage |
| `meiki abandon <id> [reason]` | Mark a todo as abandoned |
| `meiki resolve <id> [how]` | Mark a blocker as resolved |
| `meiki reopen <id>` | Restore an item to open status |
| `meiki setup` | One-time initialization |
| `meiki doctor` | Self-diagnostic checks |

## Design

- **Append-only data model** — entries are never modified in place; mutations produce new entries with `supersedes` references
- **No LLM in the binary** — all logic is deterministic Go; the AI presents meiki's output conversationally
- **Model/tool-agnostic** — works with any AI CLI tool that can run shell commands
- **XDG-compliant** — config in `~/.config/meiki/`, data in `~/.local/share/meiki/`

## Data

Entries are stored as JSONL files organized by date:

```
~/.local/share/meiki/
  entries/2026/05/2026-05-17.jsonl
  reviews/2026/05/2026-05-17.md
  state.json
```

## Build

```bash
make build       # local binary
make release     # cross-compile for macOS/Linux (amd64 + arm64)
make install     # go install with version
```

## License

MIT
