package packets

import "pggat2/lib/fed"

type AuthenticationOk struct{}

func (T *AuthenticationOk) ReadFromPacket(packet fed.Packet) bool {
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

func (T *AuthenticationOk) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthentication, 4)
	packet = packet.AppendUint32(0)
	return packet
}
