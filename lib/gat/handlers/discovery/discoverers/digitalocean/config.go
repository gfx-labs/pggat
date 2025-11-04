package digitalocean

import (
	"encoding/json"
)

type Config struct {
	APIKey         string `json:"api_key"`
	Private        bool   `json:"private,omitempty"`
	DiscoverStandby bool   `json:"discover_standby,omitempty"`

	Filter json.RawMessage `json:"filter,omitempty" caddy:"namespace=pggat.handlers.discovery.discoverers.digitalocean.filters inline_key=filter"`
}
