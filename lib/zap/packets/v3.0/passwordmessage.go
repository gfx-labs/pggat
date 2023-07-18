package packets

import (
	"pggat2/lib/zap"
)

func ReadPasswordMessage(in *zap.ReadablePacket) (string, bool) {
	if in.ReadType() != AuthenticationResponse {
		return "", false
	}
	password, ok := in.ReadString()
	if !ok {
		return "", false
	}
	return password, true
}

func WritePasswordMessage(out *zap.Packet, password string) {
	out.WriteType(AuthenticationResponse)
	out.WriteString(password)
}
