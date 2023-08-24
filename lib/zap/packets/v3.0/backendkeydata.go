package packets

import "pggat2/lib/zap"

type BackendKeyData struct {
	CancellationKey [8]byte
}

func (T *BackendKeyData) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeBackendKeyData {
		return false
	}
	packet.ReadBytes(T.CancellationKey[:])
	return true
}

func (T *BackendKeyData) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeBackendKeyData, 8)
	packet = packet.AppendBytes(T.CancellationKey[:])
	return packet
}
