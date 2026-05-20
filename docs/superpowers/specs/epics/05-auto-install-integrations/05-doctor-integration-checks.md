# Story: Doctor Integration Checks

**Epic:** Auto-Install Integrations
**Ref:** Epic 5, Story 5

## Summary

Update `meiki doctor` to verify that AI tool integrations are properly installed, not just that meiki's own directories exist.

## Scope

### New checks

Doctor adds these checks after the existing directory/config checks:

| # | Check | Pass condition | Fail message |
|---|-------|---------------|--------------|
| 7 | MEIKI.md installed | `~/.config/meiki/MEIKI.md` exists | "MEIKI.md not installed — run `meiki setup`" |
| 8+ | Per-tool instructions | Detected tool's instruction file contains `<!-- meiki:start -->` marker | "MEIKI.md not referenced in `<file>` — run `meiki setup`" |
| 9+ | Per-tool stop hook | Detected tool's hook config contains `meiki review` | "stop hook missing for `<tool>` — run `meiki setup`" |

### Tool detection reuse

Doctor uses the same tool detection logic as setup (Story 1). Only checks tools whose config directories are present. Tools not detected are reported as skipped, not failed.

### Copilot instruction check

Since Copilot reads CLAUDE.md/GEMINI.md rather than getting its own instruction injection, doctor checks that at least one of CLAUDE.md or GEMINI.md has the meiki markers when Copilot is detected. If neither does, report: "Copilot CLI detected but no instruction files configured — run `meiki setup`".

### Output format

```
✓ Data directory: ~/.local/share/meiki
✓ Entries directory: exists
✓ Reviews directory: exists
✓ State file: not present (OK)
✓ Config file: not present (using defaults)
✓ Binary on PATH: /usr/local/bin/meiki
✓ MEIKI.md: installed
✓ Claude Code: instructions ✓  stop hook ✓
✓ Gemini CLI: instructions ✓  stop hook ✓
· Copilot CLI: not detected, skipped

All checks passed.
```

### Failure example

```
✓ Data directory: ~/.local/share/meiki
...
✓ MEIKI.md: installed
✗ Claude Code: instructions ✓  stop hook missing — run `meiki setup`
· Gemini CLI: not detected, skipped
· Copilot CLI: not detected, skipped

1 issue found. Run 'meiki setup' to fix.
```

## Acceptance Criteria

- [ ] Checks `~/.config/meiki/MEIKI.md` existence
- [ ] Checks each detected tool's instruction file for meiki markers
- [ ] Checks each detected tool's hook config for meiki review hook
- [ ] Copilot check verifies at least one of CLAUDE.md/GEMINI.md has markers
- [ ] Skipped tools show as "not detected, skipped" (not failures)
- [ ] Issue count includes integration failures
- [ ] Reuses tool detection logic from Story 1
- [ ] Tests cover: all checks pass, instructions missing, hook missing, MEIKI.md missing, tool not detected, Copilot with/without other tool instructions
