package packets

import (
	"pggat2/lib/zap3"
)

func ReadAuthenticationOk(in *zap3.ReadablePacket) bool {
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

func WriteAuthenticationOk(out *zap3.Packet) {
	out.WriteType(Authentication)
	out.WriteInt32(0)
}
