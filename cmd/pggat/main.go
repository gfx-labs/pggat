package main

import (
	caddycmd "gfx.cafe/gfx/pggat/cmd"
	_ "github.com/caddyserver/caddy/v2/modules/metrics"

	_ "gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	_ "gfx.cafe/gfx/pggat/lib/gat/standard"
)

func main() {
	caddycmd.Main()
}
