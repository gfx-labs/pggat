package backend

import "pggat2/lib/pnet"

type Server interface {
	pnet.ReadWriter
}

type Backend interface {
}
