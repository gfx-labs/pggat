package sasl

import "pggat2/lib/auth/sasl/scram"

type Server interface {
	InitialResponse(bytes []byte) ([]byte, bool, error)
	Continue(bytes []byte) ([]byte, bool, error)
}

func NewServer(mechanism, username, password string) (Server, error) {
	switch mechanism {
	case scram.SHA256:
		return scram.MakeServer(mechanism, username, password)
	default:
		return nil, ErrMechanismsNotSupported
	}
}
