package eqp

import (
	"hash/maphash"

	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type PreparedStatement struct {
	packet zap.Packet
	hash   uint64
}

func ReadParse(packet zap.Packet) (destination string, preparedStatement PreparedStatement, ok bool) {
	if packet.Type() != packets.TypeParse {
		return
	}

	packet.ReadString(&destination)

	preparedStatement.packet = packet
	preparedStatement.hash = maphash.Bytes(seed, preparedStatement.packet.Payload())
	ok = true
	return
}
