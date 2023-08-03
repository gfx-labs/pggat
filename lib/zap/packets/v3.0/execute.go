package packets

import "pggat2/lib/zap"

func ReadExecute(in zap.ReadablePacket) (target string, maxRows int32, ok bool) {
	if in.ReadType() != Execute {
		return
	}
	target, ok = in.ReadString()
	if !ok {
		return
	}
	maxRows, ok = in.ReadInt32()
	if !ok {
		return
	}
	return
}

func WriteExecute(out *zap.Packet, target string, maxRows int32) {
	out.WriteType(Execute)
	out.WriteString(target)
	out.WriteInt32(maxRows)
}
