package strutil

import "strings"

// CutLeft is similar to strings.Cut but it returns "", s, false if not found
func CutLeft(s string, sep string) (before, after string, found bool) {
	before, after, found = strings.Cut(s, sep)
	if !found {
		after = before
		before = ""
	}
	return
}

// CutRight is similar to strings.Cut but it searches from the end first
func CutRight(s string, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
