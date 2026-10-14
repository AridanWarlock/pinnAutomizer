package migrations

import "embed"

// Files contains the SQL migrations shipped with the solver binary.
//
//go:embed *.sql
var Files embed.FS
