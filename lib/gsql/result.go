package gsql

import "gfx.cafe/gfx/pggat/lib/fed"

type ResultWriter interface {
	WritePacket(fed.Packet) error
}
