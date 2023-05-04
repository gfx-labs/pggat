package pnet

import "pggat2/lib/pnet/packet"

type Writer interface {
	Write() packet.Out
}
