package main

import (
	"bytes"
	"go/format"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

const (
	CODEGEN = "./codegen/protocol"
	INPUT   = "./spec/protocol"
	OUTPUT  = "./lib/gat/protocol"
)

var funcs = template.FuncMap{
	"list": func(x ...any) []any {
		return x
	},
	"camelCase": func(v string) string {
		return strcase.ToCamel(v)
	},
}

func main() {
	f, err := os.ReadDir(INPUT)
	if err != nil {
		panic(err)
	}
	t := template.Must(template.New("packets.tmpl").Funcs(funcs).ParseFiles(filepath.Join(CODEGEN, "packets.tmpl")))
	err = os.MkdirAll(OUTPUT, 0777)
	if err != nil {
		panic(err)
	}
	all := make(map[string]any)
	var out bytes.Buffer
	for _, e := range f {
		var b []byte
		b, err = os.ReadFile(filepath.Join(INPUT, e.Name()))
		if err != nil {
			panic(err)
		}
		var packets map[string]any
		err = yaml.Unmarshal(b, &packets)
		if err != nil {
			panic(err)
		}

		for k, v := range packets {
			all[k] = v
		}

		err = t.Execute(&out, packets)
		if err != nil {
			panic(err)
		}

		var fmtd []byte
		fmtd, err = format.Source(out.Bytes())
		if err != nil {
			panic(err)
		}

		// output to file
		err = os.WriteFile(filepath.Join(OUTPUT, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))+".go"), fmtd, 0777)
		if err != nil {
			panic(err)
		}

		out.Reset()
	}

	t = template.Must(template.New("mod.tmpl").Funcs(funcs).ParseFiles(filepath.Join(CODEGEN, "mod.tmpl")))
	err = t.Execute(&out, all)
	if err != nil {
		panic(err)
	}

	var fmtd []byte
	fmtd, err = format.Source(out.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filepath.Join(OUTPUT, "mod.go"), fmtd, 0777)
	if err != nil {
		panic(err)
	}
}
