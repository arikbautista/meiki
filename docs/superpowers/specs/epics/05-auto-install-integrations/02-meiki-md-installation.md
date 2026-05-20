# Story: MEIKI.md Installation

**Epic:** Auto-Install Integrations
**Ref:** Epic 5, Story 2

## Summary

Write the embedded MEIKI.md to a shared config location and inject include references into each detected tool's global instruction file.

## Scope

### Shared MEIKI.md location

Setup writes the embedded MEIKI.md content to `~/.config/meiki/MEIKI.md`. This file is the single source of truth — no per-tool copies. Overwritten on every `meiki setup` run so updates to the embedded content propagate automatically.

### Include injection

For tools that support `@path` include syntax, setup appends a marked block to the tool's global instruction file:

| Tool | Instruction file | Action |
|------|-----------------|--------|
| Claude Code | `~/.claude/CLAUDE.md` | Append include block |
| Gemini CLI | `~/.gemini/GEMINI.md` | Append include block |
| Copilot CLI | *(none)* | Skipped — reads CLAUDE.md and GEMINI.md automatically |

### Block format

The include line is wrapped in HTML comment markers for idempotent detection:

```markdown
<!-- meiki:start -->
@/home/user/.config/meiki/MEIKI.md
<!-- meiki:end -->
```

The `@` path uses the resolved absolute home directory (from `os.UserHomeDir()`), not `~`, because AI CLI tool parsers may not expand tilde in include syntax. The actual path is generated at runtime by `meiki setup`.

### Merge behavior

- **File does not exist:** Create the file with only the meiki block.
- **File exists, no markers:** Append the meiki block to the end of the file, preceded by a blank line.
- **File exists, markers present:** Replace the content between existing markers with the current include line. This handles upgrades where the include path might change.
- **File exists but unreadable:** Report an error, do not modify. Suggest `--print-only`.

### Copilot CLI

Copilot CLI reads `CLAUDE.md` and `GEMINI.md` files from `~/.claude/` and `~/.gemini/` respectively. Once either of those is configured, Copilot picks up MEIKI.md instructions automatically. No Copilot-specific instruction file modification is needed.

### Output

```
  ✓ Claude Code — instructions installed
  ✓ Gemini CLI — instructions installed
  ✓ Copilot CLI — instructions via Claude Code/Gemini CLI
```

## Acceptance Criteria

- [ ] Writes MEIKI.md to `~/.config/meiki/MEIKI.md` with embedded content
- [ ] Creates `~/.claude/CLAUDE.md` with meiki block if it doesn't exist
- [ ] Appends meiki block to existing `~/.claude/CLAUDE.md` without markers
- [ ] Replaces meiki block in existing `~/.claude/CLAUDE.md` with markers
- [ ] Same behavior for `~/.gemini/GEMINI.md`
- [ ] Does not modify any Copilot CLI instruction files
- [ ] Skips instruction injection for tools not detected
- [ ] Re-running setup does not duplicate the include block
- [ ] Unreadable instruction files produce a clear error without modification
- [ ] Tests cover: create new, append to existing, replace existing, unreadable file, tool not detected
