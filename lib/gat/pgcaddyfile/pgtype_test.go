package pgcaddyfile_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig"

	"gfx.cafe/gfx/pggat/lib/gat"
	_ "gfx.cafe/gfx/pggat/lib/gat/pgcaddyfile"
)

func adapt(t *testing.T, input string) map[string]json.RawMessage {
	t.Helper()
	adapter := caddyconfig.GetAdapter("caddyfile")
	if adapter == nil {
		t.Fatal("standard caddyfile adapter is not registered")
	}
	output, _, err := adapter.Adapt([]byte(input), nil)
	if err != nil {
		t.Fatalf("adapt config: %v", err)
	}
	var config struct {
		Apps    map[string]json.RawMessage `json:"apps"`
		Admin   json.RawMessage            `json:"admin"`
		Logging json.RawMessage            `json:"logging"`
	}
	if err := json.Unmarshal(output, &config); err != nil {
		t.Fatalf("decode adapted config: %v", err)
	}
	return map[string]json.RawMessage{
		"apps":    mustJSON(t, config.Apps),
		"admin":   config.Admin,
		"logging": config.Logging,
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal decoded value: %v", err)
	}
	return data
}

func TestPGDefaultStatLogPeriod(t *testing.T) {
	config := adapt(t, `[pg] :5433 {
    require_ssl
}
`)
	var apps map[string]json.RawMessage
	if err := json.Unmarshal(config["apps"], &apps); err != nil {
		t.Fatalf("decode apps: %v", err)
	}
	var app gat.App
	if err := json.Unmarshal(apps["pggat"], &app); err != nil {
		t.Fatalf("decode pggat app: %v", err)
	}
	if got, want := time.Duration(app.StatLogPeriod).String(), "1m0s"; got != want {
		t.Fatalf("default stat_log_period = %s, want %s", got, want)
	}
}

func TestPGBlocksAggregateAndOptionsCanAppearAfterServers(t *testing.T) {
	config := adapt(t, `[pg] :5433 {
    max_connections 50
    require_ssl
}

[pg] {
    stat_log_period 2m
}

[pg] :6432 {
    require_ssl
}
`)
	var apps map[string]json.RawMessage
	if err := json.Unmarshal(config["apps"], &apps); err != nil {
		t.Fatalf("decode apps: %v", err)
	}
	var app gat.App
	if err := json.Unmarshal(apps["pggat"], &app); err != nil {
		t.Fatalf("decode pggat app: %v", err)
	}
	if got, want := len(app.Servers), 2; got != want {
		t.Fatalf("server count = %d, want %d", got, want)
	}
	if got, want := time.Duration(app.StatLogPeriod).String(), "2m0s"; got != want {
		t.Fatalf("stat_log_period = %s, want %s", got, want)
	}
	if got, want := app.Servers[0].Listen[0].Address, ":5433"; got != want {
		t.Fatalf("first listener = %q, want %q", got, want)
	}
	if got, want := app.Servers[0].Listen[0].MaxConnections, 50; got != want {
		t.Fatalf("max connections = %d, want %d", got, want)
	}
	if got, want := app.Servers[1].Listen[0].Address, ":6432"; got != want {
		t.Fatalf("second listener = %q, want %q", got, want)
	}
}

func TestStandardCaddyfileSupportsMixedPGHTTPAndGlobalOptions(t *testing.T) {
	config := adapt(t, `[global] {
    admin off
    log {
        output stderr
    }
}

[http.server] example.com {
    respond "hello"
}

[pg] :5433 {
    require_ssl
}
`)
	var apps map[string]json.RawMessage
	if err := json.Unmarshal(config["apps"], &apps); err != nil {
		t.Fatalf("decode apps: %v", err)
	}
	for _, name := range []string{"http", "pggat"} {
		if len(apps[name]) == 0 {
			t.Errorf("apps.%s missing from adapted config", name)
		}
	}
	if len(config["admin"]) == 0 || string(config["admin"]) == "null" {
		t.Error("Caddy global admin option was not preserved")
	}
	if len(config["logging"]) == 0 || string(config["logging"]) == "null" {
		t.Error("Caddy global logging option was not preserved")
	}
}

func TestPGOptionErrorsAndLegacyUnmarkedPGBlock(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
	}{
		{name: "invalid duration", input: "[pg] {\n    stat_log_period nope\n}\n"},
		{name: "zero duration", input: "[pg] {\n    stat_log_period 0\n}\n"},
		{name: "negative duration", input: "[pg] {\n    stat_log_period -1s\n}\n"},
		{name: "extra argument", input: "[pg] {\n    stat_log_period 1m extra\n}\n"},
		{name: "nested block", input: "[pg] {\n    stat_log_period 1m {\n        ignored\n    }\n}\n"},
		{name: "unknown option", input: "[pg] {\n    unknown_option x\n}\n"},
		{name: "legacy unmarked pg block", input: ":5433 {\n    require_ssl\n}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			adapter := caddyconfig.GetAdapter("caddyfile")
			if _, _, err := adapter.Adapt([]byte(test.input), nil); err == nil {
				t.Fatal("expected adaptation error")
			}
		})
	}
}
