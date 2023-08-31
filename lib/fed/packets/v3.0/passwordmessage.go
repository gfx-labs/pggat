package packets

import (
	"pggat2/lib/fed"
)

type PasswordMessage struct {
	Password string
}

func (T *PasswordMessage) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthenticationResponse {
		return false
	}
	packet.ReadString(&T.Password)
	return true
}

func (T *PasswordMessage) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthenticationResponse, len(T.Password)+1)
	packet = packet.AppendString(T.Password)
	return packet
}
