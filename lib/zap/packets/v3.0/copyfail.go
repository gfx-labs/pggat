package packets

import "pggat2/lib/zap"

func ReadCopyFail(in zap.ReadablePacket) (string, bool) {
	if in.ReadType() != CopyFail {
		return "", false
	}
	reason, ok := in.ReadString()
	if !ok {
		return "", false
	}
	return reason, true
}

func WriteCopyFail(out *zap.Packet, reason string) {
	out.WriteType(CopyFail)
	out.WriteString(reason)
}
