package gatcaddyfile

import "gfx.cafe/gfx/pggat/lib/gat/handlers/pool/pools/basic"

var defaultPool = &basic.Factory{
	Config: basic.Transaction,
}
