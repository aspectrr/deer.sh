// Package shellutil provides shell-safe string quoting utilities.
package shellutil

import "strings"

// Quote wraps a string in POSIX single quotes, escaping any embedded
// single quotes with the '\” idiom.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
