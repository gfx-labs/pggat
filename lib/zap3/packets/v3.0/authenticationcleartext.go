package packets

import (
	"pggat2/lib/zap3"
)

func ReadAuthenticationCleartext(in *zap3.ReadablePacket) bool {
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

func WriteAuthenticationCleartext(out *zap3.Packet) {
	out.WriteType(Authentication)
	out.WriteInt32(3)
}
