// Package schemas exposes the jig manifest JSON Schema embedded in the
// binary. The schema file in this directory is the canonical copy,
// validated against at manifest load time.
package schemas

import "embed"

// FS holds the jig manifest schema.
//
//go:embed *.schema.json
var FS embed.FS
