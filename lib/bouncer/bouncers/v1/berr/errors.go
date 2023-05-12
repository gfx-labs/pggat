package berr

import (
	"errors"

	packets "pggat2/lib/zap/packets/v3.0"
)

var (
	ServerProtocolError = Server{
		Error: errors.New("protocol error"),
	}
	ServerBadPacket = Server{
		Error: errors.New("bad packet"),
	}

	ClientProtocolError = Client{
		Error: packets.ErrUnexpectedPacket,
	}
	ClientBadPacket = Client{
		Error: packets.ErrBadFormat,
	}
)
