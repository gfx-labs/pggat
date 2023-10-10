package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type AuthenticationMD5 struct {
	Salt [4]byte
}

func (T *AuthenticationMD5) ReadFrom(packet fed.PacketDecoder) error {
	if packet.Type != TypeAuthentication {
		return ErrUnexpectedPacket
	}

	var method int32
	err := packet.
		Int32(&method).
		Bytes(T.Salt[:]).
		Error
	if err != nil {
		return err
	}

	if method != 5 {
		return ErrBadFormat
	}
	return nil
}

func (T *AuthenticationMD5) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeAuthentication, 8)
	packet = packet.AppendUint32(5)
	packet = packet.AppendBytes(T.Salt[:])
	return packet
}
