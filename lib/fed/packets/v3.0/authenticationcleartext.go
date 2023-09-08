package packets

import "pggat/lib/fed"

type AuthenticationCleartext struct{}

func (T *AuthenticationCleartext) ReadFromPacket(packet fed.Packet) bool {
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

func (T *AuthenticationCleartext) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthentication, 4)
	packet = packet.AppendUint32(3)
	return packet
}
