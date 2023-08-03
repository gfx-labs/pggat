package eqp

import (
	"hash/maphash"

	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Portal struct {
	source string
	packet *zap.Packet
	hash   uint64
}

func ReadBind(in zap.ReadablePacket) (destination string, portal Portal, ok bool) {
	if in.ReadType() != packets.Bind {
		return
	}
	in2 := in
	destination, ok = in2.ReadString()
	if !ok {
		return
	}
	portal.source, ok = in2.ReadString()
	if !ok {
		return
	}

	portal.packet = zap.NewPacket()
	portal.packet.WriteType(packets.Bind)
	portal.packet.WriteBytes(in.ReadUnsafeRemaining())
	portal.hash = maphash.Bytes(seed, portal.packet.Payload())
	return
}

func (T *Portal) Done() {
	T.packet.Done()
	T.packet = nil
}
