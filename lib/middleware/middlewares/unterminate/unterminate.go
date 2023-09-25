package unterminate

import (
	"io"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/middleware"
)

// Unterminate catches the Terminate packet and returns io.EOF instead.
// Useful if you don't want to forward to the server and close the connection.
var Unterminate = unterm{}

type unterm struct{}

func (unterm) Read(_ middleware.Context, packet fed.Packet) error {
	if packet.Type() == packets.TypeTerminate {
		return io.EOF
	}
	return nil
}

func (unterm) Write(_ middleware.Context, _ fed.Packet) error {
	return nil
}

var _ middleware.Middleware = unterm{}
