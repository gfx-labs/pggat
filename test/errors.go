package test

import (
	"strings"
)

type ErrorIn struct {
	Name string
	Err  error
}

func (T ErrorIn) Error() string {
	var b strings.Builder
	b.WriteString(`Error in "`)
	b.WriteString(T.Name)
	b.WriteString("\":\n\t")

	sub := T.Err.Error()
	for _, r := range sub {
		if r == '\n' {
			b.WriteString("\n\t")
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

var _ error = ErrorIn{}

type Errors []error

func (T Errors) Error() string {
	var b strings.Builder
	for _, err := range T {
		b.WriteString(err.Error())
		b.WriteRune('\n')
	}
	return b.String()
}

var _ error = Errors{}
