package berr

import "errors"

var (
	ServerBadPacket        = MakeServer(errors.New("bad packet format"))
	ServerUnexpectedPacket = MakeServer(errors.New("unexpected packet"))

	ClientUnexpectedPacket = MakeClient(errors.New("unexpected packet"))
)
