# Epic 3: Onboarding & Distribution

**Goal:** Make meiki installable, self-diagnosing, and integrated with AI CLI tools out of the box.

**Depends on:** Epic 2 (CLI Commands) — setup and doctor validate that commands work; MEIKI.md references command syntax.

**Done when:** A user can install the binary, run `meiki setup`, paste the output into their AI tool config, run `meiki doctor` to verify, and have the AI automatically capture entries and produce briefings.

## Stories

| # | Story | Summary |
|---|-------|---------|
| 1 | [Setup Command](01-setup-command.md) | `meiki setup` — one-time init, prints config snippets |
| 2 | [Doctor Command](02-doctor-command.md) | `meiki doctor` — self-diagnostic checks |
| 3 | [MEIKI.md Instructions Pack](03-meiki-md-instructions.md) | Canonical AI instructions file shipped with the binary |
| 4 | [Stop Hook Configuration](04-stop-hook.md) | Claude Code Stop hook setup and documentation |
| 5 | [Cross-Platform Build & Release](05-build-and-release.md) | Static binaries for macOS (intel+arm) and Linux |
