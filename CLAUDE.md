# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

meiki is a local CLI tool that captures work memory during AI CLI sessions, produces daily reviews, and delivers next-morning briefings. Written in Go as a single static binary.

## Architecture

```
cmd/meiki/          # main package, cobra command tree
internal/           # core logic — no exported API
  entry/            # entry model, ULID generation, JSONL I/O
  scanner/          # open-item scanner, priority decay
  review/           # review markdown generation
  brief/            # briefing output, debouncing
  config/           # config.toml + state.json management
instructions/       # MEIKI.md — canonical AI instructions pack
docs/               # design spec, epic/story specs
```

Data paths follow XDG conventions on all platforms:
- Config: `~/.config/meiki/config.toml`
- Data: `~/.local/share/meiki/` (entries/, reviews/, state.json)

## Key Design Constraints

- **Append-only data model.** Entries are never modified in place. Mutations (close, abandon, resolve, reopen) produce new entries with a `supersedes` field referencing the prior entry. Open-item state is derived by scanning the log, not maintained as a separate index.
- **Atomic writes.** JSONL files must be opened with `O_APPEND` for concurrent write safety. Single entries fit within `PIPE_BUF`.
- **No LLM in the binary.** All "smarts" are deterministic Go code. The AI presents meiki's output conversationally; meiki itself does no generation.
- **Model/tool-agnostic.** meiki knows nothing about specific AI tools, MCP servers, or external services.

## Build & Test

```bash
go build ./cmd/meiki
go test ./...
go test ./internal/entry/    # single package
go test -run TestCapture ./internal/entry/  # single test
```

## Specs

- Design spec: `docs/superpowers/specs/2026-05-15-meiki-design.md`
- Epics & stories: `docs/superpowers/specs/epics/` — each epic has a `_epic.md` summary and numbered story specs

### Epic overview

1. **Foundation** (6 stories) — Go scaffold, data model, JSONL I/O, scanner, priority decay
2. **CLI Commands** (9 stories) — capture, abandon/resolve/reopen, open/today/recent, review, brief
3. **Onboarding & Distribution** (5 stories) — setup, doctor, MEIKI.md, stop hook, build/release

Dependencies: Epic 1 → Epic 2 → Epic 3
