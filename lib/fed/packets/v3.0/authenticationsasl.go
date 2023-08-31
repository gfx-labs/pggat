package packets

import "pggat2/lib/fed"

type AuthenticationSASL struct {
	Mechanisms []string
}

func (T *AuthenticationSASL) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	p := packet.ReadInt32(&method)
	if method != 10 {
		return false
	}
	T.Mechanisms = T.Mechanisms[:0]
	for {
		var mechanism string
		p = p.ReadString(&mechanism)
		if mechanism == "" {
			break
		}
		T.Mechanisms = append(T.Mechanisms, mechanism)
	}
	return true
}

func (T *AuthenticationSASL) IntoPacket() fed.Packet {
	size := 5
	for _, mechanism := range T.Mechanisms {
		size += len(mechanism) + 1
	}

	packet := fed.NewPacket(TypeAuthentication, size)

	packet = packet.AppendInt32(10)
	for _, mechanism := range T.Mechanisms {
		packet = packet.AppendString(mechanism)
	}
	packet = packet.AppendUint8(0)
	return packet
}
