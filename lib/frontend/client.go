package frontend

import (
	"pggat2/lib/pnet"
)

type Client interface {
	pnet.ReadWriteSender
}
