package backends

import (
	"errors"
	"fmt"

	"gfx.cafe/gfx/pggat/lib/fed"
)

func ErrUnexpectedPacket(typ fed.Type) error {
	return fmt.Errorf("unexpected packet: %c", typ)
}

var (
	ErrExpectedIdle                     = errors.New("expected server to return ReadyForQuery(IDLE)")
	ErrUnexpectedAuthenticationResponse = errors.New("unexpected authentication response")
)
