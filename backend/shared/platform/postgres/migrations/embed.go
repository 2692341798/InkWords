// Package migrations owns SQL assets that will be executed by the production migration runner.
package migrations

import "embed"

// Files keeps versioned migrations adjacent to the binary. The runner is introduced separately
// with the goose dependency so this package does not create a production migration side effect.
//
//go:embed *.sql
var Files embed.FS
