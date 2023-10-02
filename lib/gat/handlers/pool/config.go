package pool_handler

import (
	"encoding/json"

	"gfx.cafe/gfx/pggat/lib/bouncer"
)

type Config struct {
	Pooler json.RawMessage `json:"pooler" caddy:"namespace=pggat.poolers inline_key=pooler"`

	// Server connect options
	ServerAddress string          `jsonn:"server_address"`
	ServerSSLMode bouncer.SSLMode `json:"server_ssl_mode,omitempty"`
	ServerSSL     json.RawMessage `json:"server_ssl,omitempty" caddy:"namespace=pggat.ssl.clients inline_key=provider"`

	// Server routing options
	ServerUsername          string            `json:"server_username"`
	ServerPassword          string            `json:"server_password"`
	ServerDatabase          string            `json:"server_database"`
	ServerStartupParameters map[string]string `json:"server_startup_parameters"`
}
