package packets

import (
	"pggat2/lib/zap"
)

func ReadSASLInitialResponse(in zap.Inspector) (mechanism string, initialResponse []byte, ok bool) {
	in.Reset()
	if in.Type() != AuthenticationResponse {
		return
	}

	mechanism, ok = in.String()
	if !ok {
		return
	}

	var initialResponseSize int32
	initialResponseSize, ok = in.Int32()
	if !ok {
		return
	}
	if initialResponseSize == -1 {
		return
	}

	initialResponse, ok = in.UnsafeBytes(int(initialResponseSize))
	return
}

func WriteSASLInitialResponse(out zap.Builder, mechanism string, initialResponse []byte) {
	out.Reset()
	out.Type(AuthenticationResponse)
	out.String(mechanism)
	if initialResponse == nil {
		out.Int32(-1)
	} else {
		out.Int32(int32(len(initialResponse)))
		out.Bytes(initialResponse)
	}
}
