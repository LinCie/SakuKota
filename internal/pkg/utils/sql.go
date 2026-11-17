package utils

import "strings"

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLike escapes LIKE wildcards so s matches literally.
func EscapeLike(s string) string {
	return likeEscaper.Replace(s)
}
