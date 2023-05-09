package sasl

import (
	"errors"

	"pggat2/lib/auth/sasl/scram"
)

var ErrMechanismsNotSupported = errors.New("SASL mechanisms not supported")

type Client interface {
	Name() string
	InitialResponse() []byte
	Continue([]byte) ([]byte, error)
	Final([]byte) error
}

func NewClient(mechanisms []string, username, password string) (Client, error) {
	for _, mechanism := range mechanisms {
		switch mechanism {
		case scram.SHA256:
			return scram.MakeClient(mechanism, username, password)
		}
	}
	return nil, ErrMechanismsNotSupported
}
