package eqp

import (
	"hash/maphash"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
)

type PreparedStatement struct {
	packet fed.Packet
	hash   uint64
}

func ReadParse(packet fed.Packet) (destination string, preparedStatement PreparedStatement, ok bool) {
	if packet.Type() != packets.TypeParse {
		return
	}

	packet.ReadString(&destination)

	preparedStatement.packet = packet
	preparedStatement.hash = maphash.Bytes(seed, preparedStatement.packet.Payload())
	ok = true
	return
}
