package gat

import "gfx.cafe/gfx/pggat/lib/auth"

type Pooler interface {
	NewPool(creds auth.Credentials) *Pool
}
