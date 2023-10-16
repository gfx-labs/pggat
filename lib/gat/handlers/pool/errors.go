package pool

import "errors"

var (
	ErrClosed = errors.New("pools closed")
)
