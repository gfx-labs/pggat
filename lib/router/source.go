package router

import "pggat2/lib/pnet"

type Source interface {
	Handle(peer pnet.ReadWriter, write bool)
}
