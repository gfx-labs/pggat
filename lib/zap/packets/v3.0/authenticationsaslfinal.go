package packets

import "pggat2/lib/zap"

func ReadAuthenticationSASLFinal(in *zap.ReadablePacket) ([]byte, bool) {
	if in.ReadType() != Authentication {
		return nil, false
	}
	method, ok := in.ReadInt32()
	if !ok {
		return nil, false
	}
	if method != 12 {
		return nil, false
	}
	return in.ReadUnsafeRemaining(), true
}

func WriteAuthenticationSASLFinal(out *zap.Packet, resp []byte) {
	out.WriteType(Authentication)
	out.WriteInt32(12)
	out.WriteBytes(resp)
}
