package standard

import (
	// base server
	_ "gfx.cafe/gfx/pggat/lib/gat"

	// matchers
	_ "gfx.cafe/gfx/pggat/lib/gat/matchers"

	// ssl servers
	_ "gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
	_ "gfx.cafe/gfx/pggat/lib/gat/ssl/servers/x509_key_pair"

	// ssl clients
	_ "gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"

	// middlewares
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/allowed_startup_parameters"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/error"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/require_ssl"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_database"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_parameter"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_password"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_user"

	// handlers
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer_spilo"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/pool"

	// discovery
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/google_cloud_sql"
	_ "gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/zalando_operator"

	// poolers
	_ "gfx.cafe/gfx/pggat/lib/gat/poolers/session"
	_ "gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
)
