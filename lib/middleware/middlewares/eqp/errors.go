package eqp

import "errors"

var (
	ErrBadPacketFormat         = errors.New("bad packet format")
	ErrPreparedStatementExists = errors.New("prepared statement already exists")
	ErrPortalExists            = errors.New("portal already exists")
	ErrUnknownCloseTarget      = errors.New("unknown close target")
)
