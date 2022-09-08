package parse

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Command struct {
	Index     int
	Command   string
	Arguments []string
}

type reader struct {
	v string
	p int
}

var EndOfSQL = errors.New("end of sql")
var EndOfStatement = errors.New("end of statement")
var NotThisToken = errors.New("end of token")
var UnexpectedCharacter = errors.New("unexpected character")

func newUnexpectedCharacter(c rune) error {
	return fmt.Errorf("%w: '%c'", UnexpectedCharacter, c)
}

func (r *reader) nextRune() (rune, bool) {
	if r.p >= len(r.v) {
		return '-', false
	}
	c, l := utf8.DecodeRuneInString(r.v[r.p:])
	r.p += l
	return c, true
}

func (r *reader) nextComment() error {
	c, ok := r.nextRune()
	if !ok {
		return EndOfSQL
	}
	switch {
	case c == ';':
		return EndOfStatement
	case c == '-':
		// we good
	default:
		return NotThisToken
	}

	return r.nextString("\n")
}

func (r *reader) nextMultiLineComment() error {
	c, ok := r.nextRune()
	if !ok {
		return EndOfSQL
	}
	switch {
	case c == ';':
		return EndOfStatement
	case c == '*':
		// we good
	default:
		return NotThisToken
	}

	return r.nextString("*/")
}

func (r *reader) nextIdentifier() (string, error) {
	var stack strings.Builder
	for {
		c, ok := r.nextRune()
		if !ok {
			break
		}
		switch {
		case c == ';':
			return stack.String(), EndOfStatement
		case unicode.IsSpace(c):
			if stack.Len() == 0 {
				continue
			}

			// this identifier is done
			return stack.String(), nil
		case unicode.IsDigit(c):
			if stack.Len() == 0 {
				return "", newUnexpectedCharacter(c)
			}
			fallthrough
		case unicode.IsLetter(c), c == '_', c == '$':
			stack.WriteRune(c)
		case c == '-' && stack.Len() == 0:
			if r.nextComment() != nil {
				return "", newUnexpectedCharacter(c)
			}
		default:
			return "", newUnexpectedCharacter(c)
		}
	}

	return stack.String(), EndOfSQL
}

func (r *reader) nextString(delim string) error {
	di := 0
	escaping := false
	for {
		d, l := utf8.DecodeRuneInString(delim[di:])
		c, ok := r.nextRune()
		if !ok {
			return EndOfSQL
		}

		switch c {
		case d:
			di += l
			if di >= len(delim) {
				di = 0
				if !escaping {
					return nil
				}
			}
			escaping = false
		case '\\':
			escaping = true
			di = 0
		default:
			di = 0
			escaping = false
		}
	}
}

func (r *reader) nextDollarIdentifier() error {
	start := r.p
	for {
		pre := r.p
		c, ok := r.nextRune()
		if !ok {
			return EndOfSQL
		}

		switch {
		case c == ';':
			return EndOfStatement
		case unicode.IsSpace(c):
			// this identifier is done
			return NotThisToken
		case unicode.IsDigit(c):
			if start == pre {
				return NotThisToken
			}
		case unicode.IsLetter(c), c == '_':
		case c == '$':
			return nil
		default:
			return NotThisToken
		}
	}
}

func (r *reader) nextArgument() (string, error) {
	// just read everything up to spaces or the end token, being mindful of strings and end of statements
	start := r.p

	for {
		pre := r.p
		c, ok := r.nextRune()
		if !ok {
			break
		}

		switch {
		case unicode.IsSpace(c):
			if pre == start {
				start = r.p
				continue
			}

			// this argument is done
			return r.v[start:pre], nil
		case c == ';':
			return r.v[start:pre], EndOfStatement
		case c == '\'', c == '"':
			err := r.nextString(string(c))
			if err != nil {
				return r.v[start:r.p], err
			}
		case c == '$' && pre == start:
			// try the dollar string
			err := r.nextDollarIdentifier()
			if err != nil {
				if err == NotThisToken {
					err = nil
					continue
				}
				return r.v[start:r.p], err
			}

			err = r.nextString(r.v[pre:r.p])
			if err != nil {
				return r.v[start:r.p], err
			}
		case c == '-' && pre == start:
			err := r.nextComment()
			if err != nil {
				if err == NotThisToken {
					err = nil
					continue
				}
				return r.v[start:r.p], err
			}
		case c == '/':
			err := r.nextMultiLineComment()
			if err != nil {
				if err == NotThisToken {
					err = nil
					continue
				}
				return r.v[start:r.p], err
			}
		}
	}

	return r.v[start:], EndOfSQL
}

func (r *reader) nextCommand() (cmd Command, err error) {
	cmd.Index = r.p
	cmd.Command, err = r.nextIdentifier()
	if err != nil {
		if err == EndOfStatement {
			err = nil
		}
		return
	}

	for {
		var arg string
		arg, err = r.nextArgument()

		if arg != "" {
			cmd.Arguments = append(cmd.Arguments, arg)
		}

		if err != nil {
			if errors.Is(err, EndOfStatement) {
				err = nil
			}
			return
		}
	}
}

// Parse parses an sql query in a single pass. Because all we really care about is the commands, this can be very fast
// based on https://www.postgresql.org/docs/current/sql-syntax-lexical.html
func Parse(sql string) (cmds []Command, err error) {
	r := reader{
		v: sql,
	}
	for {
		var cmd Command
		cmd, err = r.nextCommand()

		if cmd.Command != "" {
			cmds = append(cmds, cmd)
		}

		if err != nil {
			if errors.Is(err, EndOfSQL) {
				err = nil
			}
			return
		}
	}
}
