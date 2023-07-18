package packets

import "pggat2/lib/zap"

func ReadAuthenticationMD5(in *zap.ReadablePacket) ([4]byte, bool) {
	if in.ReadType() != Authentication {
		return [4]byte{}, false
	}
	method, ok := in.ReadInt32()
	if !ok {
		return [4]byte{}, false
	}
	if method != 5 {
		return [4]byte{}, false
	}
	var salt [4]byte
	ok = in.ReadBytes(salt[:])
	if !ok {
		return salt, false
	}
	return salt, true
}

func WriteAuthenticationMD5(out *zap.Packet, salt [4]byte) {
	out.WriteType(Authentication)
	out.WriteUint32(5)
	out.WriteBytes(salt[:])
}
