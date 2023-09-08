package eqp

import (
	"hash/maphash"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
)

type Portal struct {
	source string
	packet fed.Packet
	hash   uint64
}

func ReadBind(in fed.Packet) (destination string, portal Portal, ok bool) {
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
