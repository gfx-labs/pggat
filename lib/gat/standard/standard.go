package standard

import (
	// base server
	_ "gfx.cafe/gfx/pggat/lib/gat"

	// matchers
	_ "gfx.cafe/gfx/pggat/lib/gat/matchers"

	// ssl servers
	_ "gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"

	// ssl clients
	_ "gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"

	// providers
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/discovery"
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/pgbouncer"
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/zalando"

	// discovery
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/discovery/discoverers/digitalocean"
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/discovery/discoverers/google_cloud_sql"
	_ "gfx.cafe/gfx/pggat/lib/gat/providers/discovery/discoverers/zalando_operator"

	// poolers
	_ "gfx.cafe/gfx/pggat/lib/gat/poolers/session"
	_ "gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
)
