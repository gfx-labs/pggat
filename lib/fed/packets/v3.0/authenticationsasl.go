package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type AuthenticationSASL struct {
	Mechanisms []string
}

func (T *AuthenticationSASL) ReadFrom(packet fed.PacketDecoder) error {
	if packet.Type != TypeAuthentication {
		return ErrUnexpectedPacket
	}

	var method int32
	p := packet.Int32(&method)
	if method != 10 {
		return ErrBadFormat
	}
	T.Mechanisms = T.Mechanisms[:0]
	for {
		var mechanism string
		p = p.String(&mechanism)
		if mechanism == "" {
			break
		}
		T.Mechanisms = append(T.Mechanisms, mechanism)
	}
	return p.Error
}

func (T *AuthenticationSASL) IntoPacket(packet fed.Packet) fed.Packet {
	size := 5
	for _, mechanism := range T.Mechanisms {
		size += len(mechanism) + 1
	}

	packet = packet.Reset(TypeAuthentication, size)

	packet = packet.AppendInt32(10)
	for _, mechanism := range T.Mechanisms {
		packet = packet.AppendString(mechanism)
	}
	packet = packet.AppendUint8(0)
	return packet
}
