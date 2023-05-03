package main

import (
	"pggat2/lib/frontend/frontends/v0"
)

func main() {
	frontend, err := frontends.NewFrontend()
	if err != nil {
		panic(err)
	}
	err = frontend.Run()
	if err != nil {
		panic(err)
	}
}
