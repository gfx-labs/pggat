package pool

import "errors"

var (
	ErrFailedToAcquirePeer = errors.New("failed to acquire peer (try increasing client_acquire_timeout?)")
)
