# Story: Data Directory & Configuration

**Epic:** Foundation
**Ref:** Design spec §5.2, §8

## Summary

Implement XDG-style path resolution for config and data directories, config.toml parsing with defaults, and data directory initialization (creating the `entries/` and `reviews/` subdirectories).

## Scope

### Path resolution

- Config: `~/.config/meiki/config.toml`
- Data: `~/.local/share/meiki/` with subdirectories `entries/` and `reviews/`
- Respect `XDG_CONFIG_HOME` and `XDG_DATA_HOME` environment variables if set

### Config model

```toml
[ui]
brief_max_open_todos = 20   # default
open_scan_days = 30          # default
stale_triage_days = 3        # default
```

- Config file is optional — all keys have sensible defaults
- Unknown keys are silently ignored (forward-compatible)

### Data directory creation

- `EnsureDataDir()` creates `~/.local/share/meiki/`, `entries/`, and `reviews/` if they don't exist
- Idempotent — safe to call on every command invocation

## Acceptance Criteria

- [ ] `ConfigDir()` returns `~/.config/meiki` by default, respects `XDG_CONFIG_HOME`
- [ ] `DataDir()` returns `~/.local/share/meiki` by default, respects `XDG_DATA_HOME`
- [ ] `LoadConfig()` returns defaults when no config file exists
- [ ] `LoadConfig()` parses a config.toml and overrides only specified keys
- [ ] `EnsureDataDir()` creates the full directory tree idempotently
- [ ] Tests cover XDG override, missing file, partial config, and full config scenarios
