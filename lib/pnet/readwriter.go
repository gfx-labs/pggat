package pnet

import "pggat2/lib/pnet/packet"

type ReadWriter interface {
	Reader
	Writer
}

type ReadWriteSender interface {
	ReadWriter
	packet.Sender
}
