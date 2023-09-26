package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	_ "gfx.cafe/gfx/pggat/contrib/caddy"
)

func main() {
	caddycmd.Main()
}
