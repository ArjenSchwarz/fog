package lib

import (
	"regexp"
	"strings"
)

// GlobToRegex converts a glob pattern (where * matches any sequence of characters)
// into a compiled regular expression anchored at both ends. All regex metacharacters
// in the input are escaped so that only * is treated as a wildcard.
func GlobToRegex(pattern string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(pattern)
	regexStr := "^" + strings.ReplaceAll(escaped, `\*`, ".*") + "$"
	return regexp.MustCompile(regexStr)
}
