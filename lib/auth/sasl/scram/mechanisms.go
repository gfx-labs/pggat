package scram

import "errors"

var ErrUnsupportedMethod = errors.New("unsupported SCRAM method")

const (
	SHA256 = "SCRAM-SHA-256"
)

var Mechanisms = []string{
	SHA256,
}
