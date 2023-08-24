package packets

import (
	"pggat2/lib/zap"
)

type PasswordMessage struct {
	Password string
}

func (T *PasswordMessage) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeAuthenticationResponse {
		return false
	}
	packet.ReadString(&T.Password)
	return true
}

func (T *PasswordMessage) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthenticationResponse, len(T.Password)+1)
	packet = packet.AppendString(T.Password)
	return packet
}
