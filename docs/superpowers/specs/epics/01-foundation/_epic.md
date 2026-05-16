# Epic 1: Foundation

**Goal:** Build the core data layer and project skeleton that all CLI commands depend on.

**Scope:** Go module initialization, entry model, JSONL I/O with atomic appends, data directory management, configuration parsing, state file, open-item scanner with priority decay logic.

**Done when:** A test can write entries to JSONL files, read them back, derive open-item state from a log containing mutations, and correctly identify stale/overdue items — all without any CLI wiring.

## Stories

| # | Story | Summary |
|---|-------|---------|
| 1 | [Project Scaffolding](01-project-scaffolding.md) | Go module, directory layout, cobra skeleton with subcommand stubs |
| 2 | [Data Directory & Configuration](02-data-directory-and-config.md) | XDG path resolution, config.toml parsing, data dir creation |
| 3 | [Entry Model & JSONL I/O](03-entry-model-and-jsonl-io.md) | Entry struct, ULID generation, atomic JSONL append, reader |
| 4 | [State File Management](04-state-file.md) | state.json read/write for brief and review timestamps |
| 5 | [Open-Item Scanner](05-open-item-scanner.md) | Derive open todos/blockers from entry log, handle supersedes chains |
| 6 | [Priority Decay & Stale Detection](06-priority-decay.md) | Overdue calculation, stale-item triage classification |

## Dependencies

None — this epic is the foundation for everything else.
