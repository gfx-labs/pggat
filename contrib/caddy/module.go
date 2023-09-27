package caddy

import "encoding/json"

type ServerModule struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config,omitempty"`
}
