// Package instructions provides embedded instruction files for meiki.
package instructions

import _ "embed"

// MeikiMD contains the canonical AI instructions file content.
//
//go:embed MEIKI.md
var MeikiMD string
