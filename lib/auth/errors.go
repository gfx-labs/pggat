package auth

import "errors"

var (
	ErrMethodNotSupported        = errors.New("auth method not supported")
	ErrFailed                    = errors.New("auth failed")
	ErrSASLMechanismNotSupported = errors.New("SASL mechanism not supported")
	ErrSASLComplete              = errors.New("SASL Complete")
)
