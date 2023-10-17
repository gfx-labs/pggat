package hybrid

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

type Middleware struct {
	buf    Buffer
	bufEnc fed.Encoder
	bufDec fed.Decoder

	w int
}

func NewMiddleware() *Middleware {
	m := new(Middleware)
	m.bufEnc.Reset(&m.buf)
	m.bufDec.Reset(&m.buf)
	return m
}

func (T *Middleware) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	if err := T.bufEnc.Next(packet.Type(), packet.Length()); err != nil {
		return nil, err
	}
	if err := packet.WriteTo(&T.bufEnc); err != nil {
		return nil, err
	}
	if err := T.bufEnc.Flush(); err != nil {
		return nil, err
	}
	if err := T.bufDec.Next(packet.Type() != 0); err != nil {
		return nil, err
	}
	p := fed.PendingPacket{
		Decoder: &T.bufDec,
	}
	return p, nil
}

func (T *Middleware) WritePacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeErrorResponse:
		var p packets.ErrorResponse
		if err := fed.ToConcrete(&p, packet); err != nil {
			return nil, err
		}
		for _, field := range p {
			switch field.Code {
			case 'C':
				if perror.Code(field.Value) == perror.ReadOnlySqlTransaction {
					return nil, ErrReadOnly{}
				}
			}
		}
		return &p, nil
	}
	T.w++
	return packet, nil
}

func (T *Middleware) Reset() {
	T.buf.Reset()
	T.bufEnc.Reset(&T.buf)
	T.bufDec.Reset(&T.buf)
	T.w = 0
}

var _ fed.Middleware = (*Middleware)(nil)
