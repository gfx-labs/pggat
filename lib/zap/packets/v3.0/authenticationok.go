package packets

import "pggat2/lib/zap"

func ReadAuthenticationOk(in zap.ReadablePacket) bool {
	if in.ReadType() != Authentication {
		return false
	}
	method, ok := in.ReadInt32()
	if !ok {
		return false
	}
	if method != 0 {
		return false
	}
	return true
}

func WriteAuthenticationOk(out *zap.Packet) {
	out.WriteType(Authentication)
	out.WriteInt32(0)
}
