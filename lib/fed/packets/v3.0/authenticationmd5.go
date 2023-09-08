package packets

import "pggat/lib/fed"

type AuthenticationMD5 struct {
	Salt [4]byte
}

func (T *AuthenticationMD5) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	p := packet.ReadInt32(&method)
	if method != 5 {
		return false
	}
	p = p.ReadBytes(T.Salt[:])
	return true
}

func (T *AuthenticationMD5) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthentication, 8)
	packet = packet.AppendUint32(5)
	packet = packet.AppendBytes(T.Salt[:])
	return packet
}
