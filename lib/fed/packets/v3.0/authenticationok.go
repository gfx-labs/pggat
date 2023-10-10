package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type AuthenticationOk struct{}

func (T *AuthenticationOk) ReadFrom(packet fed.PacketDecoder) error {
	if packet.Type != TypeAuthentication {
		return ErrUnexpectedPacket
	}

	var method int32
	err := packet.Int32(&method).Error
	if err != nil {
		return err
	}

	if method != 0 {
		return ErrBadFormat
	}
	return nil
}

func (T *AuthenticationOk) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeAuthentication, 4)
	packet = packet.AppendUint32(0)
	return packet
}
