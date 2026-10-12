package caddycmd

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	_ "gfx.cafe/gfx/pggat/lib/gat/pgcaddyfile"
)

func TestLoadCaddyfile(t *testing.T) {
	const pgConfig = "[global] {\n\tadmin off\n}\n[pg] :5433 {\n\tmax_connections 42\n}\n"
	const jsonConfig = `{"admin":{"disabled":true},"apps":{"pggat":{"servers":[]}}}`
	for _, tc := range []struct {
		name, filename, adapter, input string
		defaultFile                    bool
		wantErr                        bool
	}{
		{name: "default", filename: "Caddyfile", input: pgConfig, defaultFile: true},
		{name: "prefix", filename: "Caddyfile.prod", input: pgConfig},
		{name: "suffix", filename: "pool.Caddyfile", input: pgConfig},
		{name: "lowercase", filename: "pool.caddyfile", input: pgConfig},
		{name: "explicit adapter", filename: "pool.conf", adapter: "caddyfile", input: pgConfig},
		{name: "native JSON", filename: "Caddyfile.json", input: jsonConfig},
		{name: "missing default", defaultFile: true},
		{name: "removed adapter", filename: "pool.conf", adapter: "gatfile", input: pgConfig, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tc.filename != "" {
				if err := os.WriteFile(tc.filename, []byte(tc.input), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			filename := tc.filename
			if tc.defaultFile {
				filename = ""
				resolved, err := configFileWithRespectToDefault(nil, "")
				if err != nil || resolved != tc.filename {
					t.Fatalf("default filename = %q, %v, want %q", resolved, err, tc.filename)
				}
			}
			got, loaded, err := loadConfigWithLogger(nil, filename, tc.adapter)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected removed adapter to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if loaded != tc.filename {
				t.Fatalf("loaded filename = %q, want %q", loaded, tc.filename)
			}
			if tc.filename == "" {
				if len(got) != 0 {
					t.Fatalf("missing default loaded %s", got)
				}
				return
			}
			if tc.input == jsonConfig {
				if string(got) != jsonConfig {
					t.Fatalf("native JSON changed: %s", got)
				}
				return
			}
			var cfg struct {
				Admin struct{ Disabled bool }
				Apps  map[string]json.RawMessage
			}
			if err := json.Unmarshal(got, &cfg); err != nil {
				t.Fatal(err)
			}
			var pg struct {
				Servers []struct {
					Listen []struct {
						Address        string
						MaxConnections int `json:"max_connections"`
					}
				}
			}
			if err := json.Unmarshal(cfg.Apps["pggat"], &pg); err != nil {
				t.Fatal(err)
			}
			if !cfg.Admin.Disabled || len(pg.Servers) != 1 || len(pg.Servers[0].Listen) != 1 ||
				pg.Servers[0].Listen[0].Address != ":5433" || pg.Servers[0].Listen[0].MaxConnections != 42 {
				t.Fatalf("Caddy globals or PG server options lost: %s", got)
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	for i, tc := range []struct {
		input     string
		expect    map[string]string
		shouldErr bool
	}{
		{
			input: `KEY=value`,
			expect: map[string]string{
				"KEY": "value",
			},
		},
		{
			input: `
				KEY=value
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":       "value",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				KEY=value
				INVALID KEY=asdf
				OTHER_KEY=Some Value
			`,
			shouldErr: true,
		},
		{
			input: `
				KEY=value
				SIMPLE_QUOTED="quoted value"
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":           "value",
				"SIMPLE_QUOTED": "quoted value",
				"OTHER_KEY":     "Some Value",
			},
		},
		{
			input: `
				KEY=value
				NEWLINES="foo
	bar"
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":       "value",
				"NEWLINES":  "foo\n\tbar",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				KEY=value
				ESCAPED="\"escaped quotes\"
here"
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":       "value",
				"ESCAPED":   "\"escaped quotes\"\nhere",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				export KEY=value
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":       "value",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				=value
				OTHER_KEY=Some Value
			`,
			shouldErr: true,
		},
		{
			input: `
				EMPTY=
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"EMPTY":     "",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				EMPTY=""
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"EMPTY":     "",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				KEY=value
				#OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY": "value",
			},
		},
		{
			input: `
				KEY=value
				COMMENT=foo bar  # some comment here
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":       "value",
				"COMMENT":   "foo bar",
				"OTHER_KEY": "Some Value",
			},
		},
		{
			input: `
				KEY=value
				WHITESPACE=   foo 
				OTHER_KEY=Some Value
			`,
			shouldErr: true,
		},
		{
			input: `
				KEY=value
				WHITESPACE="   foo bar "
				OTHER_KEY=Some Value
			`,
			expect: map[string]string{
				"KEY":        "value",
				"WHITESPACE": "   foo bar ",
				"OTHER_KEY":  "Some Value",
			},
		},
	} {
		actual, err := parseEnvFile(strings.NewReader(tc.input))
		if err != nil && !tc.shouldErr {
			t.Errorf("Test %d: Got error but shouldn't have: %v", i, err)
		}
		if err == nil && tc.shouldErr {
			t.Errorf("Test %d: Did not get error but should have", i)
		}
		if tc.shouldErr {
			continue
		}
		if !reflect.DeepEqual(tc.expect, actual) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.expect, actual)
		}
	}
}
