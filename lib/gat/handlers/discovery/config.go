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

	Pooler json.RawMessage `json:"pooler" caddy:"namespace=pggat.poolers inline_key=pooler"`

	ServerSSLMode bouncer.SSLMode `json:"server_ssl_mode"`
	ServerSSL     json.RawMessage `json:"server_ssl" caddy:"namespace=pggat.ssl.clients inline_key=provider"`

	ServerStartupParameters map[string]string `json:"server_startup_parameters,omitempty"`
}
