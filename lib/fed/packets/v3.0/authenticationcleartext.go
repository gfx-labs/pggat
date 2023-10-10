package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type AuthenticationCleartext struct{}

func (T *AuthenticationCleartext) ReadFrom(decoder *fed.Decoder) error {
	if decoder.Type() != TypeAuthentication {
		return ErrUnexpectedPacket
	}

	method, err := decoder.Int32()
	if err != nil {
		return err
	}

	if method != 3 {
		return ErrBadFormat
	}
	return nil
}

func (T *AuthenticationCleartext) WriteTo(encoder *fed.Encoder) error {
	if err := encoder.Next(TypeAuthentication, 4); err != nil {
		return err
	}
	return encoder.Uint32(3)
}
