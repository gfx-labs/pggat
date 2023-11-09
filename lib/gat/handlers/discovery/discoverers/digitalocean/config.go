package digitalocean

import "gfx.cafe/gfx/pggat/lib/util/strutil"

type Config struct {
	APIKey  string `json:"api_key"`
	Private bool   `json:"private,omitempty"`

	Filter strutil.Matcher `json:"filter,omitempty"`
}
