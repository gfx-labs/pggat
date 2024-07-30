package unterminate

import (
	"io"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

// Unterminate catches the Terminate packet and returns io.EOF instead.
// Useful if you don't want to forward to the server and close the connection.
var Unterminate fed.Middleware = unterm{}

type unterm struct{}

func (unterm) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (unterm) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	if packet.Type() == packets.TypeTerminate {
		return packet, io.EOF
	}
	return packet, nil
}

func (unterm) WritePacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

func (unterm) PostWrite() (fed.Packet, error) {
	return nil, nil
}
