# Story: Cross-Platform Build & Release

**Epic:** Onboarding & Distribution
**Ref:** Design spec §9

## Summary

Set up cross-compilation for static binaries and a release pipeline for distributing meiki.

## Scope

### Target platforms

- macOS amd64 (Intel)
- macOS arm64 (Apple Silicon)
- Linux amd64
- Linux arm64

### Build

- `CGO_ENABLED=0 go build` for static binaries with no runtime dependencies
- Version injected via `-ldflags "-X main.version=..."` at build time
- Makefile or Goreleaser config for reproducible builds

### Release artifacts

- Binaries named `meiki-<os>-<arch>` (or tarballs with the binary inside)
- GitHub Releases as the primary distribution channel
- SHA256 checksums for each artifact

### Install paths (from spec)

1. Primary: Homebrew tap (`brew install meiki`) — tap repo setup, formula
2. Secondary: `go install github.com/arikbautista/meiki/cmd/meiki@latest`
3. Tertiary: prebuilt binaries from GitHub Releases

### Homebrew tap

- Separate tap repo or `homebrew-meiki` repo with a formula
- Formula downloads the correct prebuilt binary for the platform
- `brew install` → `meiki --version` works

## Acceptance Criteria

- [ ] `make build` (or equivalent) produces static binaries for all 4 targets
- [ ] Version is baked into the binary at build time
- [ ] `go install` works from the public module path
- [ ] GitHub Release automation produces checksummed artifacts
- [ ] Homebrew formula installs a working binary on macOS
