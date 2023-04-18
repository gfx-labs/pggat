package main

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"strings"

	_ "embed"
)

// errors copied from https://www.postgresql.org/docs/current/errcodes-appendix.html
//
//go:embed errors.txt
var errors string

func main() {
	var out strings.Builder
	lines := strings.Split(errors, "\n")
	for _, line := range lines {
		kv := strings.Split(line, "\t")
		out.WriteString(fmt.Sprintf("%s = \"%s\"\n", strcase.ToCamel(kv[1]), kv[0]))
	}
	fmt.Println(out.String())
}
