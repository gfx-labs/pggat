package backends

import "errors"

var (
	ErrBadFormat        = errors.New("bad packet format")
	ErrUnexpectedPacket = errors.New("unexpected packet")
)
