package strutil

import (
	"unicode/utf8"
)

type Matcher string

func (T Matcher) Matches(haystack string) bool {
	matcher := T
	for !(len(matcher) == 0 && len(haystack) == 0) {
		e, size := utf8.DecodeRuneInString(string(matcher))
		if e == utf8.RuneError {
			return false
		}
		matcher = matcher[size:]

		if e == '*' {
			if len(matcher) == 0 {
				return true
			}

			_, i := utf8.DecodeLastRuneInString(haystack)
			for {
				h := haystack[len(haystack)-i:]

				if matcher.Matches(h) {
					return true
				}

				c, size := utf8.DecodeLastRuneInString(haystack[:len(haystack)-i])
				if c == utf8.RuneError {
					return false
				}
				i += size
			}
		}

		c, size := utf8.DecodeRuneInString(haystack)
		if c == utf8.RuneError {
			return false
		}
		haystack = haystack[size:]

		if c != e {
			return false
		}
	}

	return true
}
