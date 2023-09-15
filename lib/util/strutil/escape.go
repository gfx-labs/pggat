package strutil

import (
	"strings"
	"unicode/utf8"
)

func Escape(str string, char rune) string {
	size := 0
	escape := false
	// check if it has any bad characters
	for _, r := range str {
		size += utf8.RuneLen(r)
		if r == char || r == '\\' {
			size += 1
			escape = true
		}
	}
	if !escape {
		return str
	}

	var b strings.Builder
	b.Grow(size)
	for _, r := range str {
		if char == r || r == '\\' {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
