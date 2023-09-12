package parsers

import (
	"strings"

	"pggat/test/testfile"
	"pggat/test/testfile/parser"
)

func NewlineOrEOF(ctx *parser.Context) (struct{}, bool) {
	c, ok := parser.Any(ctx)
	if !ok || c == '\n' {
		return struct{}{}, true
	}
	return struct{}{}, false
}

func Whitespace(ctx *parser.Context) (struct{}, bool) {
	var n int
	for {
		_, ok := parser.SingleOf(ctx, func(r rune) bool {
			switch r {
			case ' ', '\t', '\r', '\n', '\v', '\f':
				return true
			default:
				return false
			}
		})
		if !ok {
			break
		}
		n++
	}
	return struct{}{}, n > 0
}

func Identifier(ctx *parser.Context) (string, bool) {
	var b strings.Builder
	for {
		c, ok := parser.SingleOf(ctx, func(r rune) bool {
			return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
		})
		if !ok {
			break
		}
		b.WriteRune(c)
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

func Opcode(ctx *parser.Context) (testfile.Opcode, bool) {
	ident, ok := Identifier(ctx)
	if !ok {
		return 0, false
	}
	return testfile.OpcodeFromString(ident)
}

func StringArgument(ctx *parser.Context) (string, bool) {
	// open quote
	_, ok := parser.Single(ctx, '"')
	if !ok {
		return "", false
	}

	var b strings.Builder
	for {
		c, ok := parser.Any(ctx)
		if !ok {
			return "", false
		}
		if c == '"' {
			break
		}
		b.WriteRune(c)
	}

	return b.String(), true
}

func BoolArgument(ctx *parser.Context) (bool, bool) {
	ident, ok := Identifier(ctx)
	if !ok {
		return false, false
	}
	switch ident {
	case "false":
		return false, true
	case "true":
		return true, true
	default:
		return false, false
	}
}

func FloatArgument(ctx *parser.Context) (float64, bool) {
	return 0, false // TODO(garet)
}

func IntArgument(ctx *parser.Context) (int, bool) {
	return 0, false // TODO(garet)
}

func NumberArgument(ctx *parser.Context) (any, bool) {
	if arg, ok := parser.Try(ctx, FloatArgument); ok {
		return arg, true
	}
	if arg, ok := parser.Try(ctx, IntArgument); ok {
		return arg, true
	}
	return nil, false
}

func Argument(ctx *parser.Context) (any, bool) {
	if arg, ok := parser.Try(ctx, StringArgument); ok {
		return arg, true
	}
	if arg, ok := parser.Try(ctx, BoolArgument); ok {
		return arg, true
	}
	if arg, ok := parser.Try(ctx, NumberArgument); ok {
		return arg, true
	}
	return nil, false
}

func Arguments(ctx *parser.Context) ([]any, bool) {
	var args []any
	for {
		arg, ok := parser.Try(ctx, Argument)
		if !ok {
			break
		}
		args = append(args, arg)
		Whitespace(ctx)
	}
	return args, len(args) != 0
}

func Instruction(ctx *parser.Context) (testfile.Instruction, bool) {
	Whitespace(ctx)
	opcode, ok := Opcode(ctx)
	if !ok {
		return testfile.Instruction{}, false
	}

	Whitespace(ctx)
	arguments, _ := Arguments(ctx)

	Whitespace(ctx)
	_, ok = NewlineOrEOF(ctx)
	if !ok {
		return testfile.Instruction{}, false
	}

	return testfile.Instruction{
		Op:   opcode,
		Args: arguments,
	}, true
}
