# Story: Stop Hook Installation

**Epic:** Auto-Install Integrations
**Ref:** Epic 5, Story 3

## Summary

Install a session-end hook running `meiki review --silent` in each detected tool's native hook format.

## Scope

### Hook definitions

**Claude Code** — merged into `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "meiki review --silent"}]
    }]
  }
}
```

**Gemini CLI** — merged into `~/.gemini/settings.json`:

```json
{
  "hooks": {
    "SessionEnd": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "meiki review --silent"}]
    }]
  }
}
```

**Copilot CLI** — written as `~/.copilot/hooks/meiki-review.json`:

```json
{
  "event": "sessionEnd",
  "command": "meiki review --silent"
}
```

### Merge behavior for settings.json (Claude Code, Gemini CLI)

1. **File does not exist:** Create it with just the hooks block.
2. **File exists, valid JSON:**
   - If no `hooks` key: add it with the meiki hook.
   - If `hooks` key exists but no matching event key (`Stop` / `SessionEnd`): add the event array.
   - If event key exists: scan the array for an existing entry whose command contains `meiki review`. If found, skip. If not found, append the meiki hook entry to the array.
3. **File exists, invalid JSON:** Report an error, do not modify. Suggest `--print-only`.

### Merge behavior for Copilot CLI

Copilot hooks are individual files in `~/.copilot/hooks/`. Create the `hooks/` directory if it doesn't exist. Write `meiki-review.json` if it doesn't exist. If it already exists, skip.

### Hook detection for idempotency

The meiki hook is identified by its command string containing `meiki review`. This avoids duplicates when the hook already exists with slightly different formatting.

### Output

```
  ✓ Claude Code — stop hook installed
  ✓ Gemini CLI — stop hook installed
  ✓ Copilot CLI — stop hook installed
```

## Acceptance Criteria

- [ ] Creates `~/.claude/settings.json` with hook when file doesn't exist
- [ ] Merges hook into existing `~/.claude/settings.json` preserving other settings
- [ ] Does not duplicate hook when already present in `~/.claude/settings.json`
- [ ] Same three behaviors for `~/.gemini/settings.json` with `SessionEnd` event
- [ ] Creates `~/.copilot/hooks/` directory if missing
- [ ] Writes `~/.copilot/hooks/meiki-review.json` when it doesn't exist
- [ ] Skips Copilot hook file when it already exists
- [ ] Reports clear error on unparseable settings.json without modifying it
- [ ] Skips hook installation for tools not detected
- [ ] Tests cover: create new, merge into existing with other hooks, merge into existing with meiki hook already present, invalid JSON, tool not detected
