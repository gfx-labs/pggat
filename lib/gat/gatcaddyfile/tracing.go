package gatcaddyfile

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/otel_tracing"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"log/slog"
)

const (
	OtelTracing = "otel"
)

func init() {
	RegisterDirective(Tracing, OtelTracing, func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := otel_tracing.NewModule()

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "service_name":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected service name value")
				}

				module.ServiceName = d.Val()
			case "service_namespace":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected service namespace value")
				}

				module.ServiceNamespace = d.Val()
			case "endpoint":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected endpoint value")
				}

				module.Endpoint = d.Val()
			case "batch_timeout":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected batch timeout value")
				}

				dur, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return nil, err
				}
				module.BatchTimout = &dur
			case "sample_rate":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected sample rate value")
				}

				module.SamplerRate = d.Val()
			case "log_level":
				if !d.NextArg() {
					return nil, fmt.Errorf("expected log level value")
				}

				var level slog.Level
				if err := level.UnmarshalText([]byte(d.Val())); err != nil {
					return nil, err
				}
				module.LogLevel = level
			}
		}

		return module, nil
	})
}
