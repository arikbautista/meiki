# Story: Setup Command

**Epic:** Onboarding & Distribution
**Ref:** Design spec §6.4, §9

## Summary

Implement `meiki setup` — the one-time-per-machine initialization command that creates the data directory and prints actionable snippets for the user to integrate meiki with their AI tools.

## Scope

### Usage

```
meiki setup
```

### Behavior

1. Creates `~/.local/share/meiki/` with `entries/` and `reviews/` subdirectories (via `EnsureDataDir()`)
2. Creates `~/.config/meiki/` if it doesn't exist
3. Prints to stdout:
   - A success message confirming directory creation
   - The full `MEIKI.md` content (from the embedded instructions pack) with instructions to paste/include it in `~/.claude/CLAUDE.md` and equivalent files for other AI tools
   - The JSON snippet for the Claude Code Stop hook (to paste into `~/.claude/settings.json`)
   - A reminder to run `meiki doctor` to verify

### Idempotent

- Safe to re-run. Re-creates directories if missing, always prints the snippets.
- Does NOT overwrite existing config files — only creates directories.

### Embedded content

The `MEIKI.md` content and Stop hook JSON are embedded in the binary (Go `embed` package) from the `instructions/` directory, not generated at runtime.

## Acceptance Criteria

- [ ] Creates data and config directories idempotently
- [ ] Prints the full MEIKI.md content ready for copy-paste
- [ ] Prints the Stop hook JSON snippet ready for copy-paste
- [ ] Instructions specify where to paste each snippet
- [ ] Re-running produces identical output
- [ ] Exit code 0 on success, 2 on filesystem errors
