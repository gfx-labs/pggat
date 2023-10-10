package main

import (
	_ "embed"

	"gfx.cafe/util/temple"
	"gfx.cafe/util/temple/lib/prayer"
)

func main() {
	var obj any
	temple.RegisterTemplateDir("templates")
	temple.ReadObjectFile(&obj, "protocol.yaml")
	temple.Prepare(&prayer.Go{
		Input:   "packets",
		Obj:     obj,
		Package: "packets",
		Output:  "out/packets.go",
	})
	if err := temple.Pray(); err != nil {
		panic(err)
	}
}
