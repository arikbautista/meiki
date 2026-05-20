# Story: Tool Detection

**Epic:** Auto-Install Integrations
**Ref:** Epic 5, Story 1

## Summary

Implement a tool detection layer that identifies which AI CLI tools are installed on the current machine by checking for their config directories.

## Scope

### Supported tools

| Tool | Config directory | Detection signal |
|------|-----------------|------------------|
| Claude Code | `~/.claude/` | Directory exists |
| Gemini CLI | `~/.gemini/` | Directory exists |
| Copilot CLI | `~/.copilot/` | Directory exists |

### Detection interface

Each tool is represented as a struct with its detection logic, config paths, and integration methods. This is an internal abstraction — not an exported API.

```go
type toolIntegration struct {
    Name           string
    ConfigDir      string
    InstructionFile string
    HookConfigPath string
    Detected       bool
}
```

A `detectTools()` function returns a slice of `toolIntegration` values with `Detected` set based on directory existence. The home directory is resolved via `os.UserHomeDir()`.

### Output format

Setup prints detection results before performing any configuration:

```
Detected tools:
  ✓ Claude Code
  ✓ Gemini CLI
  · Copilot CLI — not detected, skipped
```

If no tools are detected:

```
No AI CLI tools detected. Install Claude Code, Gemini CLI, or Copilot CLI,
then run `meiki setup` again.
```

Setup still creates meiki's own directories even when no tools are detected.

### Testability

Detection uses the home directory as a parameter (not hardcoded `os.UserHomeDir()`) so tests can use temp directories with selective tool config dirs present.

## Acceptance Criteria

- [ ] Detects Claude Code via `~/.claude/` directory presence
- [ ] Detects Gemini CLI via `~/.gemini/` directory presence
- [ ] Detects Copilot CLI via `~/.copilot/` directory presence
- [ ] Returns all three tools with correct `Detected` status
- [ ] Prints detection summary to stdout
- [ ] Handles no tools detected — prints message, still creates meiki directories
- [ ] Home directory is injectable for testing
- [ ] Tests cover: all detected, none detected, partial detection
