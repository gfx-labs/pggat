package packets

import "pggat2/lib/zap"

func ReadParameterStatus(in *zap.ReadablePacket) (key, value string, ok bool) {
	if in.ReadType() != ParameterStatus {
		return
	}
	key, ok = in.ReadString()
	if !ok {
		return
	}
	value, ok = in.ReadString()
	if !ok {
		return
	}
	return
}

func WriteParameterStatus(out *zap.Packet, key, value string) {
	out.WriteType(ParameterStatus)
	out.WriteString(key)
	out.WriteString(value)
}
