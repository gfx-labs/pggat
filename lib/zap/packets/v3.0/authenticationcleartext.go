package packets

import "pggat2/lib/zap"

func ReadAuthenticationCleartext(in *zap.ReadablePacket) bool {
	if in.ReadType() != Authentication {
		return false
	}
	method, ok := in.ReadInt32()
	if !ok {
		return false
	}
	if method != 3 {
		return false
	}
	return true
}

func WriteAuthenticationCleartext(out *zap.Packet) {
	out.WriteType(Authentication)
	out.WriteInt32(3)
}
