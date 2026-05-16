# Story: Project Scaffolding

**Epic:** Foundation
**Ref:** Design spec §4.1, §9

## Summary

Initialize the Go module and directory layout. Wire up a cobra command tree with stubs for all v1 subcommands so that subsequent stories can implement them independently.

## Scope

- `go mod init` with module path `github.com/arikbautista/meiki`
- Create directory structure: `cmd/meiki/`, `internal/{entry,scanner,review,brief,config}/`, `instructions/`
- `cmd/meiki/main.go` — cobra root command with version flag
- Subcommand stubs: `capture`, `brief`, `review`, `open`, `today`, `recent`, `abandon`, `resolve`, `reopen`, `setup`, `doctor`
- Each stub prints "not implemented" and exits 1
- Exit code conventions: 0 success, 1 user error, 2 internal error

## Acceptance Criteria

- [ ] `go build ./cmd/meiki` produces a static binary
- [ ] `go test ./...` passes (even if no real tests yet)
- [ ] `meiki --version` prints the version
- [ ] Every v1 subcommand is registered and shows in `meiki --help`
- [ ] Each subcommand stub exits 1 with a "not implemented" message

## Out of Scope

- Actual command logic (later stories)
- Build automation, CI, Makefile (Epic 3)
