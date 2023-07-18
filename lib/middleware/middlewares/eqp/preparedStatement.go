package eqp

import (
	"hash/maphash"

	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type PreparedStatement struct {
	packet *zap.Packet
	hash   uint64
}

func ReadParse(in *zap.ReadablePacket) (destination string, preparedStatement PreparedStatement, ok bool) {
	if in.ReadType() != packets.Parse {
		return
	}
	in2 := *in
	destination, ok = in2.ReadString()
	if !ok {
		return
	}

	preparedStatement.packet = zap.NewPacket()
	preparedStatement.packet.WriteType(packets.Parse)
	preparedStatement.packet.WriteBytes(in.ReadUnsafeRemaining())
	preparedStatement.hash = maphash.Bytes(seed, preparedStatement.packet.Payload())
	return
}

func (T *PreparedStatement) Done() {
	T.packet.Done()
	T.packet = nil
}
