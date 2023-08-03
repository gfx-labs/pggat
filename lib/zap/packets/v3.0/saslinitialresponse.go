package packets

import "pggat2/lib/zap"

func ReadSASLInitialResponse(in zap.ReadablePacket) (mechanism string, initialResponse []byte, ok bool) {
	if in.ReadType() != AuthenticationResponse {
		return
	}

	mechanism, ok = in.ReadString()
	if !ok {
		return
	}

	var initialResponseSize int32
	initialResponseSize, ok = in.ReadInt32()
	if !ok {
		return
	}
	if initialResponseSize == -1 {
		return
	}

	initialResponse, ok = in.ReadUnsafeBytes(int(initialResponseSize))
	return
}

func WriteSASLInitialResponse(out *zap.Packet, mechanism string, initialResponse []byte) {
	out.WriteType(AuthenticationResponse)
	out.WriteString(mechanism)
	if initialResponse == nil {
		out.WriteInt32(-1)
	} else {
		out.WriteInt32(int32(len(initialResponse)))
		out.WriteBytes(initialResponse)
	}
}
