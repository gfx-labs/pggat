package unterminate

import (
	"io"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

// Unterminate catches the Terminate packet and returns io.EOF instead.
// Useful if you don't want to forward to the server and close the connection.
var Unterminate = unterm{}

type unterm struct {
	middleware.Nil
}

func (unterm) Read(_ middleware.Context, packet zap.Packet) error {
	if packet.Type() == packets.TypeTerminate {
		return io.EOF
	}
	return nil
}

var _ middleware.Middleware = unterm{}
