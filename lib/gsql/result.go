package gsql

import "pggat2/lib/fed"

type ResultWriter interface {
	WritePacket(packet fed.Packet) error
	Done() bool
}
