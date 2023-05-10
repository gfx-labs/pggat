package pnet

import (
	"io"

	"pggat2/lib/pnet/packet"
)

type Writer interface {
	io.ByteWriter

	Write() packet.Out
	Send(packet.Type, []byte) error
}
