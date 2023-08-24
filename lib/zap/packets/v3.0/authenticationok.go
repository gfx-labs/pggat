package packets

import "pggat2/lib/zap"

type AuthenticationOk struct{}

func (T *AuthenticationOk) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	packet.ReadInt32(&method)
	if method != 0 {
		return false
	}
	return true
}

func (T *AuthenticationOk) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthentication)
	packet = packet.AppendUint32(0)
	return packet
}
