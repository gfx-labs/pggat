package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	_ "gfx.cafe/gfx/pggat/contrib/caddy"
	_ "gfx.cafe/gfx/pggat/lib/gat/modules/cloud_sql_discovery"
	_ "gfx.cafe/gfx/pggat/lib/gat/modules/digitalocean_discovery"
	_ "gfx.cafe/gfx/pggat/lib/gat/modules/pgbouncer"
	_ "gfx.cafe/gfx/pggat/lib/gat/modules/zalando"
	_ "gfx.cafe/gfx/pggat/lib/gat/modules/zalando_operator_discovery"
)

func main() {
	caddycmd.Main()
}
