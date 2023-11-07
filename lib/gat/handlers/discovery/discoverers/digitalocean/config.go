package digitalocean

import "gfx.cafe/gfx/pggat/lib/util/strutil"

type Priority struct {
	Filter strutil.Matcher `json:"filter"`
	Value  int             `json:"value"`
}

type Config struct {
	APIKey  string `json:"api_key"`
	Private bool   `json:"private,omitempty"`

	Filter   strutil.Matcher `json:"filter,omitempty"`
	Priority []Priority      `json:"priority,omitempty"`
}
