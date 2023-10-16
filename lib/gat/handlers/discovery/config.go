package discovery

import (
	"encoding/json"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/util/dur"
)

type Config struct {
	// ReconcilePeriod is how often the module should check for changes. 0 = disable
	ReconcilePeriod dur.Duration `json:"reconcile_period"`

	Discoverer json.RawMessage `json:"discoverer" caddy:"namespace=pggat.handlers.discovery.discoverers inline_key=discoverer"`

	Pool json.RawMessage `json:"pool" caddy:"namespace=pggat.handlers.pool.pools inline_key=pool"`

	ServerSSLMode bouncer.SSLMode `json:"server_ssl_mode,omitempty"`
	ServerSSL     json.RawMessage `json:"server_ssl,omitempty" caddy:"namespace=pggat.ssl.clients inline_key=provider"`

	ServerStartupParameters map[string]string `json:"server_startup_parameters,omitempty"`
}
