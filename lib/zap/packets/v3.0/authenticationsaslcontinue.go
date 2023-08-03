package packets

import "pggat2/lib/zap"

func ReadAuthenticationSASLContinue(in zap.ReadablePacket) ([]byte, bool) {
	if in.ReadType() != Authentication {
		return nil, false
	}
	method, ok := in.ReadInt32()
	if !ok {
		return nil, false
	}
	if method != 11 {
		return nil, false
	}
	return in.ReadUnsafeRemaining(), true
}

func WriteAuthenticationSASLContinue(out *zap.Packet, resp []byte) {
	out.WriteType(Authentication)
	out.WriteInt32(11)
	out.WriteBytes(resp)
}
