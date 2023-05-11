package middleware

import "pggat2/lib/pnet/packet"

type Middleware interface {
	Write(in packet.In) (forward bool, err error)
	Read(in packet.In) (forward bool, err error)
}
