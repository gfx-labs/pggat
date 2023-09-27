package gat

import "gfx.cafe/gfx/pggat/lib/fed"

type Matcher interface {
	Matches(conn fed.Conn) bool
}
