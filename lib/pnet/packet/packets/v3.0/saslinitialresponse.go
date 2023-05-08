package packets

import "pggat2/lib/pnet/packet"

func ReadSASLInitialResponse(in packet.In) (mechanism string, initialResponse []byte, ok bool) {
	in.Reset()
	if in.Type() != packet.AuthenticationResponse {
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

	initialResponse = make([]byte, int(initialResponseSize))
	ok = in.Bytes(initialResponse[:])
	if !ok {
		return
	}
	return
}

func WriteSASLInitialResponse(out packet.Out, mechanism string, initialResponse []byte) {
	out.Reset()
	out.Type(packet.AuthenticationResponse)
	out.String(mechanism)
	if initialResponse == nil {
		out.Int32(-1)
	} else {
		out.Int32(int32(len(initialResponse)))
		out.Bytes(initialResponse)
	}
}
