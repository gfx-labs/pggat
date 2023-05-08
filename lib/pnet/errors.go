package pnet

import "pggat2/lib/perror"

var ErrBadPacketFormat = perror.New(
	perror.FATAL,
	perror.ProtocolViolation,
	"Bad packet format",
)

var ErrProtocolError = perror.New(
	perror.FATAL,
	perror.ProtocolViolation,
	"Unexpected packet",
)
