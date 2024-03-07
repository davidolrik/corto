// Package migrations embeds the SQL migration files so the binary is
// self-contained and migrations can run from anywhere.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
