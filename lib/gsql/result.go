package gsql

import "pggat/lib/fed"

type ResultWriter interface {
	WritePacket(packet fed.Packet) error
	Done() bool
}
