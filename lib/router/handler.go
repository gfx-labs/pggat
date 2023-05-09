package router

import (
	"pggat2/lib/pnet"
)

type Handler interface {
	Next() pnet.ReadWriter
}
