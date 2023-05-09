package router

import (
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
)

type Router interface {
	Transaction(peer pnet.ReadWriter) perror.Error
}
