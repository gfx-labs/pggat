package packets

import (
	"pggat2/lib/zap"
)

func ReadReadyForQuery(in *zap.ReadablePacket) (byte, bool) {
	if in.ReadType() != ReadyForQuery {
		return 0, false
	}
	state, ok := in.ReadUint8()
	if !ok {
		return 0, false
	}
	return state, true
}

func WriteReadyForQuery(out *zap.Packet, state uint8) {
	out.WriteType(ReadyForQuery)
	out.WriteUint8(state)
}
