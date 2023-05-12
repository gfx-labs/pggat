package unterminate

import (
	"io"

	"pggat2/lib/mw2"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

var Unterminate = unterm{}

type unterm struct {
	mw2.Nil
}

func (unterm) Read(_ mw2.Context, in zap.In) error {
	if in.Type() == packets.Terminate {
		return io.EOF
	}
	return nil
}

var _ mw2.Middleware = unterm{}
