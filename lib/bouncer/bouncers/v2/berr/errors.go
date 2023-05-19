package berr

import "errors"

var (
	ServerBadPacket        = MakeServer(errors.New("bad packet format from server"))
	ServerUnexpectedPacket = MakeServer(errors.New("unexpected packet from server"))

	ClientUnexpectedPacket = MakeClient(errors.New("unexpected packet from client"))
)
