package sqlite

import "strings"

// Returning builds a suffix from trusted schema column names, not user input.
func Returning(columns ...string) string {
	return "RETURNING " + strings.Join(columns, ",")
}
