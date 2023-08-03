package packets

import "pggat2/lib/zap"

func ReadBackendKeyData(in zap.ReadablePacket) ([8]byte, bool) {
	if in.ReadType() != BackendKeyData {
		return [8]byte{}, false
	}
	var cancellationKey [8]byte
	ok := in.ReadBytes(cancellationKey[:])
	if !ok {
		return cancellationKey, false
	}
	return cancellationKey, true
}

func WriteBackendKeyData(out *zap.Packet, cancellationKey [8]byte) {
	out.WriteType(BackendKeyData)
	out.WriteBytes(cancellationKey[:])
}
