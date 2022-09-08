package parse

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Command struct {
	Index     int
	Command   string
	Arguments []string
}

type reader struct {
	v []rune
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
	c := r.v[r.p]
	r.p += 1
	return c, true
}

func (r *reader) nextComment() (string, error) {
	var stack strings.Builder
	c, ok := r.nextRune()
	if !ok {
		return "", EndOfSQL
	}
	stack.WriteRune(c)
	switch {
	case c == ';':
		return stack.String(), EndOfStatement
	case c == '-':
		// we good
	default:
		return stack.String(), NotThisToken
	}

	_, err := r.nextString([]rune{'\n'})
	return "", err
}

func (r *reader) nextMultiLineComment() (string, error) {
	var stack strings.Builder
	c, ok := r.nextRune()
	if !ok {
		return "", EndOfSQL
	}
	stack.WriteRune(c)
	switch {
	case c == ';':
		return stack.String(), EndOfStatement
	case c == '*':
		// we good
	default:
		return stack.String(), NotThisToken
	}

	_, err := r.nextString([]rune("*/"))
	return "", err
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
			if _, err := r.nextComment(); err != nil {
				return "", newUnexpectedCharacter(c)
			}
		default:
			return "", newUnexpectedCharacter(c)
		}
	}

	return stack.String(), EndOfSQL
}

func (r *reader) nextString(delim []rune) (string, error) {
	var stack strings.Builder
	di := 0
	escaping := false
	for {
		c, ok := r.nextRune()
		if !ok {
			return stack.String(), EndOfSQL
		}

		stack.WriteRune(c)
		switch c {
		case delim[di]:
			di += 1
			if di >= len(delim) {
				di = 0
				if !escaping {
					return stack.String(), nil
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

func (r *reader) nextDollarIdentifier() (string, error) {
	var stack strings.Builder
	stack.WriteRune('$')
	for {
		c, ok := r.nextRune()
		if !ok {
			return stack.String(), EndOfSQL
		}

		switch {
		case c == ';':
			return stack.String(), EndOfStatement
		case unicode.IsSpace(c):
			// this identifier is done
			return stack.String(), NotThisToken
		case unicode.IsDigit(c):
			if stack.Len() == 0 {
				stack.WriteRune(c)
				return stack.String(), NotThisToken
			}
			fallthrough
		case unicode.IsLetter(c), c == '_':
			stack.WriteRune(c)
		case c == '$':
			stack.WriteRune(c)
			return stack.String(), nil
		default:
			stack.WriteRune(c)
			return stack.String(), NotThisToken
		}
	}
}

func (r *reader) nextArgument() (string, error) {
	// just read everything up to spaces or the end token, being mindful of strings and end of statements
	var stack strings.Builder

	for {
		c, ok := r.nextRune()
		if !ok {
			break
		}

		switch {
		case unicode.IsSpace(c):
			if stack.Len() == 0 {
				continue
			}

			// this argument is done
			return stack.String(), nil
		case c == ';':
			return stack.String(), EndOfStatement
		case c == '\'', c == '"':
			stack.WriteRune(c)
			str, err := r.nextString([]rune{c})
			stack.WriteString(str)
			if err != nil {
				return stack.String(), err
			}
		case c == '$' && stack.Len() == 0:
			// try the dollar string
			delim, err := r.nextDollarIdentifier()
			stack.WriteString(delim)
			if err != nil {
				if errors.Is(err, NotThisToken) {
					err = nil
					continue
				}
				return stack.String(), err
			}

			str, err := r.nextString([]rune(delim))
			stack.WriteString(str)
			if err != nil {
				return stack.String(), err
			}
		case c == '-' && stack.Len() == 0:
			comment, err := r.nextComment()
			if err != nil {
				stack.WriteRune('-')
				stack.WriteString(comment)
				if errors.Is(err, NotThisToken) {
					err = nil
					continue
				}
				return stack.String(), err
			}
		case c == '/':
			comment, err := r.nextMultiLineComment()
			if err != nil {
				stack.WriteRune('/')
				stack.WriteString(comment)
				if errors.Is(err, NotThisToken) {
					err = nil
					continue
				}
				return stack.String(), err
			}
		default:
			stack.WriteRune(c)
		}
	}

	return stack.String(), EndOfSQL
}

func (r *reader) nextCommand() (cmd Command, err error) {
	cmd.Index = r.p
	cmd.Command, err = r.nextIdentifier()
	if err != nil {
		if errors.Is(err, EndOfStatement) {
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
	r := &reader{
		v: []rune(sql),
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
