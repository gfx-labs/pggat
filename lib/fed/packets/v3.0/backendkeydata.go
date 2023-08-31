package packets

import "pggat2/lib/fed"

type BackendKeyData struct {
	CancellationKey [8]byte
}

func (T *BackendKeyData) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeBackendKeyData {
		return false
	}
	packet.ReadBytes(T.CancellationKey[:])
	return true
}

func (T *BackendKeyData) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeBackendKeyData, 8)
	packet = packet.AppendBytes(T.CancellationKey[:])
	return packet
}
