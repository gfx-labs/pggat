package pnet

import "pggat2/lib/pnet/packet"

type Reader interface {
	Read() (packet.In, error)
	ReadUntyped() (packet.In, error)
}
