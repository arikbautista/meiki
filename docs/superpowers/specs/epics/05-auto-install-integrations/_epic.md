# Epic 5: Auto-Install Integrations

**Goal:** Make `meiki setup` automatically configure AI CLI tools so meiki works out of the box — no manual copy-paste of snippets.

**Problem:** The current `meiki setup` prints MEIKI.md content and a stop hook snippet, then tells the user to paste them into tool-specific config files. This requires manual configuration per tool, per machine. Users who skip or forget the paste step get no briefings and no automatic reviews.

**Approach:** Enhance `meiki setup` to auto-detect installed AI CLI tools (Claude Code, Gemini CLI, Copilot CLI), write MEIKI.md to a shared config location, inject include references into each tool's instruction file, and install session-end hooks in each tool's native format. All operations merge safely into existing config files and are idempotent. Flags provide `--dry-run`, `--print-only`, and `--uninstall` modes. `meiki doctor` is updated to verify integrations.

**Done when:** A user can install meiki, run `meiki setup`, and have every detected AI CLI tool automatically load MEIKI.md instructions and run `meiki review --silent` on session end — with zero manual file editing.

## Stories

| # | Story | Summary |
|---|-------|---------|
| 1 | [Tool Detection](01-tool-detection.md) | Detect installed AI CLI tools by config directory presence |
| 2 | [MEIKI.md Installation](02-meiki-md-installation.md) | Write MEIKI.md to shared location, inject include references into instruction files |
| 3 | [Stop Hook Installation](03-stop-hook-installation.md) | Install session-end hooks in each tool's native format |
| 4 | [CLI Flags](04-cli-flags.md) | Add `--dry-run`, `--print-only`, and `--uninstall` flags to setup |
| 5 | [Doctor Integration Checks](05-doctor-integration-checks.md) | Update doctor to verify AI tool integrations |

## Dependencies

Epic 3 (Onboarding & Distribution) — replaces the manual snippet workflow from the original setup command.
