package gat

import "gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"

type FrontendAcceptOptions = frontends.AcceptOptions

type Endpoint struct {
	Network string
	Address string

	AcceptOptions FrontendAcceptOptions
}
