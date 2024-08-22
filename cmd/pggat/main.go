package main

import (
	"context"
	caddycmd "gfx.cafe/gfx/pggat/cmd"
	"gfx.cafe/util/go/gotel"
	_ "github.com/caddyserver/caddy/v2/modules/metrics"

	_ "gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	_ "gfx.cafe/gfx/pggat/lib/gat/standard"
)

func main() {
	fn, _ := gotel.InitTracing(context.Background(), gotel.WithServiceName("pggat"))
	defer fn(context.Background())

	caddycmd.Main()
}
