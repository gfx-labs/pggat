package packets

import (
	"pggat2/lib/zap"
)

func ReadReadyForQuery(in zap.In) (byte, bool) {
	in.Reset()
	if in.Type() != ReadyForQuery {
		return 0, false
	}
	state, ok := in.Uint8()
	if !ok {
		return 0, false
	}
	return state, true
}

func WriteReadyForQuery(out zap.Out, state uint8) {
	out.Reset()
	out.Type(ReadyForQuery)
	out.Uint8(state)
}
