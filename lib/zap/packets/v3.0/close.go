package packets

import "pggat2/lib/zap"

func ReadClose(in *zap.ReadablePacket) (which uint8, target string, ok bool) {
	if in.ReadType() != Close {
		return
	}
	which, ok = in.ReadUint8()
	if !ok {
		return
	}
	target, ok = in.ReadString()
	if !ok {
		return
	}
	return
}

func WriteClose(out *zap.Packet, which uint8, target string) {
	out.WriteType(Close)
	out.WriteUint8(which)
	out.WriteString(target)
}
