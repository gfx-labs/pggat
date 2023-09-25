package gat

import "pggat/lib/bouncer/frontends/v0"

type FrontendAcceptOptions = frontends.AcceptOptions

type Endpoint struct {
	Network string
	Address string

	AcceptOptions FrontendAcceptOptions
}
