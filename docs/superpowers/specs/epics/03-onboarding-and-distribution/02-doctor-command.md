# Story: Doctor Command

**Epic:** Onboarding & Distribution
**Ref:** Design spec §6.4

## Summary

Implement `meiki doctor` — a self-diagnostic that verifies the installation is healthy and reports what it finds.

## Scope

### Usage

```
meiki doctor
```

### Checks performed

| Check | Pass | Fail |
|-------|------|------|
| Data directory exists and is writable | `~/.local/share/meiki/` exists | Directory missing or not writable |
| `entries/` subdirectory exists | Present | Missing |
| `reviews/` subdirectory exists | Present | Missing |
| `state.json` is valid (if it exists) | Parses correctly | Malformed JSON |
| `meiki` is on PATH | Which returns a path | Not found (unlikely since user is running it, but confirms PATH setup) |
| Config file parses (if it exists) | Valid TOML | Parse error with details |

### Output format

```
✓ Data directory: ~/.local/share/meiki/
✓ Entries directory: exists
✓ Reviews directory: exists
✓ State file: valid
✓ Config file: not present (using defaults)
✓ Binary on PATH: /usr/local/bin/meiki

All checks passed.
```

Or with failures:

```
✓ Data directory: ~/.local/share/meiki/
✗ Entries directory: missing — run 'meiki setup' to create
✓ Reviews directory: exists
...

1 issue found. Run 'meiki setup' to fix.
```

### Exit codes

- 0 if all checks pass
- 1 if any check fails

## Acceptance Criteria

- [ ] Runs all documented checks and reports results
- [ ] Passes on a healthy install (after `meiki setup`)
- [ ] Fails with clear guidance when directories are missing
- [ ] Detects malformed state.json and config.toml
- [ ] Exit code reflects overall health
