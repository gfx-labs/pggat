package strutil

import (
	"strings"
	"unicode/utf8"
)

func Escape(str, sequence string) string {
	var b strings.Builder
	b.WriteString(sequence)
	for len(str) > 0 {
		if strings.HasPrefix(str, sequence) {
			b.WriteByte('\\')
			b.WriteString(sequence)
			str = str[len(sequence):]
			continue
		}
		if strings.HasPrefix(str, "\\") {
			b.WriteString("\\\\")
			str = str[1:]
			continue
		}
		r, size := utf8.DecodeRuneInString(str)
		if r == utf8.RuneError {
			return ""
		}
		b.WriteRune(r)
		str = str[size:]
	}
	b.WriteString(sequence)
	return b.String()
}
