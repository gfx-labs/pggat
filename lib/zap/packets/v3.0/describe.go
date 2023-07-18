package packets

import "pggat2/lib/zap"

func ReadDescribe(in *zap.ReadablePacket) (which uint8, target string, ok bool) {
	if in.ReadType() != Describe {
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

func WriteDescribe(out *zap.Packet, which uint8, target string) {
	out.WriteType(Describe)
	out.WriteUint8(which)
	out.WriteString(target)
}
