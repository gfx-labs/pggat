package packets

import "gfx.cafe/gfx/pggat/lib/perror"

var (
	ErrBadFormat = perror.New(
		perror.FATAL,
		perror.ProtocolViolation,
		"Bad packet format",
	)
	ErrUnexpectedPacket = perror.New(
		perror.FATAL,
		perror.ProtocolViolation,
		"unexpected packet",
	)
)
