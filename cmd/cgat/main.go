package main

import "pggat2/lib/frontend/frontends/v0"

func main() {
	fe, err := frontends.NewFrontend()
	if err != nil {
		panic(err)
	}
	err = fe.Run()
	if err != nil {
		panic(err)
	}
}
