# Story: CLI Flags

**Epic:** Auto-Install Integrations
**Ref:** Epic 5, Story 4

## Summary

Add `--dry-run`, `--print-only`, and `--uninstall` flags to `meiki setup`.

## Scope

### Flags

| Flag | Behavior |
|------|----------|
| `--dry-run` | Detect tools, show what would be written or modified, but do not touch any files. For existing files, print a diff-style preview of changes. |
| `--print-only` | Print snippets to stdout for manual paste — the original `meiki setup` behavior. No file writes, no directory creation. |
| `--uninstall` | Remove all meiki integrations from detected tools. Preserve user data. |

### No flags (default)

Auto-detect tools and install everything. Create meiki directories, write MEIKI.md, inject includes, install hooks.

### `--dry-run` output

```
Detected tools:
  ✓ Claude Code
  ✓ Gemini CLI

Dry run — no files will be modified.

Would write: ~/.config/meiki/MEIKI.md
Would append to: ~/.claude/CLAUDE.md
  + <!-- meiki:start -->
  + @/home/user/.config/meiki/MEIKI.md
  + <!-- meiki:end -->
Would merge into: ~/.claude/settings.json
  + hooks.Stop: meiki review --silent
Would append to: ~/.gemini/GEMINI.md
  + <!-- meiki:start -->
  + @/home/user/.config/meiki/MEIKI.md
  + <!-- meiki:end -->
Would merge into: ~/.gemini/settings.json
  + hooks.SessionEnd: meiki review --silent
```

### `--print-only` output

Same as the current `meiki setup` behavior — prints MEIKI.md content and hook snippets for each detected tool, formatted for manual paste.

### `--uninstall` behavior

1. Remove `<!-- meiki:start -->` ... `<!-- meiki:end -->` block from each detected tool's instruction file. If the file becomes empty after removal, delete it.
2. Remove the meiki hook entry from Claude Code and Gemini CLI `settings.json`. Parse the JSON, remove the hook entry matching `meiki review`, write back. If the hooks object or event array becomes empty, remove those keys. If the file becomes `{}`, delete it.
3. Delete `~/.copilot/hooks/meiki-review.json`. Remove `~/.copilot/hooks/` directory if empty.
4. Delete `~/.config/meiki/MEIKI.md`.
5. Do **not** delete meiki data directories (`~/.local/share/meiki/`), config directory (`~/.config/meiki/`), or `config.toml`.

### Output for `--uninstall`

```
Removing meiki integrations...
  ✓ Claude Code — instructions removed, stop hook removed
  ✓ Gemini CLI — instructions removed, stop hook removed
  · Copilot CLI — not detected, skipped

Removed ~/.config/meiki/MEIKI.md
Data directory preserved: ~/.local/share/meiki/
```

### Flag conflicts

`--dry-run`, `--print-only`, and `--uninstall` are mutually exclusive. If more than one is provided, exit with an error: "flags --dry-run, --print-only, and --uninstall are mutually exclusive".

## Acceptance Criteria

- [ ] `--dry-run` shows all changes without modifying any files
- [ ] `--print-only` prints snippets to stdout, writes no files, creates no directories
- [ ] `--uninstall` removes include blocks from instruction files
- [ ] `--uninstall` removes meiki hooks from settings.json files
- [ ] `--uninstall` deletes `~/.copilot/hooks/meiki-review.json`
- [ ] `--uninstall` deletes `~/.config/meiki/MEIKI.md`
- [ ] `--uninstall` preserves data directories and config.toml
- [ ] `--uninstall` reports what was removed
- [ ] Combining mutually exclusive flags produces a clear error
- [ ] No flags runs the full auto-install flow
- [ ] Tests cover: dry-run output, print-only output, uninstall with integrations present, uninstall with nothing to remove, flag conflicts
