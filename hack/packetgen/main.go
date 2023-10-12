package main

import (
	_ "embed"
	"fmt"

	"gfx.cafe/util/temple"
	"gfx.cafe/util/temple/lib/prayer"
)

func main() {
	var idx int
	var obj any
	temple.RegisterTemplateDir("templates")
	temple.ReadObjectFile(&obj, "protocol.yaml")
	temple.RegisterFunc("temp", func() string {
		idx++
		return fmt.Sprintf("temp%d", idx)
	})
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
