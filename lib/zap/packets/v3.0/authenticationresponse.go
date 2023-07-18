package packets

import "pggat2/lib/zap"

func ReadAuthenticationResponse(in *zap.ReadablePacket) ([]byte, bool) {
	if in.ReadType() != AuthenticationResponse {
		return nil, false
	}
	return in.ReadUnsafeRemaining(), true
}

func WriteAuthenticationResponse(out *zap.Packet, resp []byte) {
	out.WriteType(AuthenticationResponse)
	out.WriteBytes(resp)
}
