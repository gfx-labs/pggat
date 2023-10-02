package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	_ "gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"

	_ "gfx.cafe/gfx/pggat/lib/gat/standard"
)

func main() {
	caddycmd.Main()
}
