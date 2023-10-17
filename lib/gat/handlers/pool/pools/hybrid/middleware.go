package hybrid

import (
	"log"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

type Middleware struct{}

func (T *Middleware) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

func (T *Middleware) WritePacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeErrorResponse:
		var p packets.ErrorResponse
		if err := fed.ToConcrete(&p, packet); err != nil {
			return nil, err
		}
		log.Printf("%#v", p)
		return &p, nil
	}
	return packet, nil
}

var _ fed.Middleware = (*Middleware)(nil)
