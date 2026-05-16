# Story: Stop Hook Configuration

**Epic:** Onboarding & Distribution
**Ref:** Design spec §4.3, §7.2

## Summary

Document and ship the Claude Code Stop hook configuration. This is primarily a documentation and `meiki setup` output concern — the hook itself is a one-line shell command.

## Scope

### Hook definition

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

- Fires on every Claude Code session end
- Runs `meiki review --silent` — updates today's review markdown without stdout output
- Safe to fire repeatedly (review generation is idempotent)
- If meiki is not on PATH, the hook fails silently (Claude Code doesn't surface hook failures to the user by default)

### What this story covers

- The JSON snippet is included in `meiki setup` output with clear instructions for where to paste it
- The snippet includes context about what it does and that it's optional (belt-and-suspenders with MEIKI.md)
- Test that `meiki review --silent` exits 0 with no stdout when called as a hook would call it

## Acceptance Criteria

- [ ] `meiki setup` prints the hook JSON snippet
- [ ] Instructions explain where to paste it (`~/.claude/settings.json`)
- [ ] `meiki review --silent` produces no stdout and exits 0
- [ ] Documentation notes this is optional if MEIKI.md instructions are followed
