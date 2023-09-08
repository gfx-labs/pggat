package unterminate

import (
	"io"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/middleware"
)

// Unterminate catches the Terminate packet and returns io.EOF instead.
// Useful if you don't want to forward to the server and close the connection.
var Unterminate = unterm{}

type unterm struct {
	middleware.Nil
}

func (unterm) Read(_ middleware.Context, packet fed.Packet) error {
	if packet.Type() == packets.TypeTerminate {
		return io.EOF
	}
	return nil
}

var _ middleware.Middleware = unterm{}
