package packets

import "pggat2/lib/zap"

type AuthenticationCleartext struct{}

func (T *AuthenticationCleartext) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	packet.ReadInt32(&method)
	if method != 3 {
		return false
	}
	return true
}

func (T *AuthenticationCleartext) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthentication, 4)
	packet = packet.AppendUint32(3)
	return packet
}
