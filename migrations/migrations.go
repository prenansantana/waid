// Package migrations exposes the SQL migration files as an embedded filesystem.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
