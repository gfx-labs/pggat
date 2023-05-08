package packets

import "pggat2/lib/pnet/packet"

func ReadReadyForQuery(in packet.In) (byte, bool) {
	in.Reset()
	if in.Type() != packet.ReadyForQuery {
		return 0, false
	}
	state, ok := in.Uint8()
	if !ok {
		return 0, false
	}
	return state, true
}

func WriteReadyForQuery(out packet.Out, state uint8) {
	out.Reset()
	out.Type(packet.ReadyForQuery)
	out.Uint8(state)
}
