package frontend

import "pggat2/lib/pnet"

type Client interface {
	pnet.ReadWriter
}

type Frontend interface {
	Run() error
}
