package pool

import "gfx.cafe/gfx/pggat/lib/auth"

type WithCredentials struct {
	*Pool
	Credentials auth.Credentials
}
