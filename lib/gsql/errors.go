package gsql

import "errors"

var (
	ErrResultTooBig       = errors.New("got too many rows for result")
	ErrExtraFields        = errors.New("received unexpected fields")
	ErrResultMustBeNonNil = errors.New("result must be non nil")
	ErrUnexpectedType     = errors.New("unexpected result type")
	ErrTypedMismatch      = errors.New("tried to read typed packet as untyped or untyped packet as typed")
)
