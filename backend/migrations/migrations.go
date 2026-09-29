// Package migrations embeds the SQL migration files.
//
// Files are named NNNN_description.up.sql (four digits). See docs/architecture.md.
package migrations

import "embed"

// FS holds all *.up.sql files.
//
//go:embed *.up.sql
var FS embed.FS
