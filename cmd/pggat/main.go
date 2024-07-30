package main

import (
	"context"
	caddycmd "gfx.cafe/gfx/pggat/cmd"
	_ "gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	_ "gfx.cafe/gfx/pggat/lib/gat/standard"
	"gfx.cafe/util/go/gotel"
)

func main() {
	fn, _ := gotel.InitTracing(context.Background(), gotel.WithServiceName("pggat"))
	defer fn(context.Background())

	caddycmd.Main()
}
