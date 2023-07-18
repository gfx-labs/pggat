package unterminate

import (
	"io"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

var Unterminate = unterm{}

type unterm struct {
	middleware.Nil
}

func (unterm) Read(_ middleware.Context, packet *zap.Packet) error {
	if packet.ReadType() == packets.Terminate {
		return io.EOF
	}
	return nil
}

var _ middleware.Middleware = unterm{}
