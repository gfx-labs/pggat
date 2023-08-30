package eqp

import (
	"hash/maphash"

	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Portal struct {
	source string
	packet zap.Packet
	hash   uint64
}

func ReadBind(in zap.Packet) (destination string, portal Portal, ok bool) {
	if in.Type() != packets.TypeBind {
		return
	}
	p := in.ReadString(&destination)
	p = p.ReadString(&portal.source)

	portal.packet = in
	portal.hash = maphash.Bytes(seed, portal.packet.Payload())
	ok = true
	return
}
