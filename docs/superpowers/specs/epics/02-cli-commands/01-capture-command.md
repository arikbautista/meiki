# Story: Capture Command

**Epic:** CLI Commands
**Ref:** Design spec §6.1, §6.2

## Summary

Implement `meiki capture` — the primary write path for creating entries. Supports all five entry types and all optional fields including `--closes` for completing todos via achievements.

## Scope

### Usage

```
meiki capture <type> "<content>" [options]

Types: achievement, learning, blocker, todo, idea

Options:
  --project <name>         Project/repo context
  --tags <a,b,c>           Comma-separated tags
  --priority <level>       tomorrow|this-week|someday (todo only)
  --due <YYYY-MM-DD>       Due date (todo only)
  --closes <id>            Todo id this achievement completes (achievement only)
  --external-ref <ref>     External reference (e.g., "jira:ENG-1234")
```

### Behavior

- Validates type is in the closed set
- Validates type-specific options: `--priority` and `--due` only allowed on `todo`; `--closes` only allowed on `achievement`
- When `--closes <id>` is used: validates the referenced entry exists and is an open todo
- Sets `source: "cli"` on all entries
- Auto-detects `project` from the current working directory basename if `--project` is not specified
- Prints the new entry's ULID to stdout on success
- Diagnostics (validation errors, file I/O errors) go to stderr

### Default priority

- `todo` entries default to `priority: "this-week"` if `--priority` is not specified
- `status` defaults to `"open"` for `todo` and `blocker` types

## Acceptance Criteria

- [ ] `meiki capture todo "do the thing"` writes a valid entry and prints the id
- [ ] `meiki capture achievement --closes <id> "did the thing"` writes an achievement and the referenced todo is no longer listed by `meiki open`
- [ ] `--closes` rejects non-existent ids and non-open todos
- [ ] `--priority` and `--due` are rejected on non-todo types
- [ ] `--closes` is rejected on non-achievement types
- [ ] Invalid types produce exit code 1 with a clear error on stderr
- [ ] Entry is written to the correct daily JSONL file (today's date)
- [ ] Project auto-detection uses cwd basename when `--project` is not provided
- [ ] `--tags` parses comma-separated values into a string array
