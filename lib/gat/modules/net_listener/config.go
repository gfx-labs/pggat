package net_listener

import "gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"

type Config struct {
	Network       string
	Address       string
	AcceptOptions frontends.AcceptOptions
}
