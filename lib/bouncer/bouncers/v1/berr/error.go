package berr

import "pggat2/lib/perror"

type Error interface {
	IsServer() bool
	IsClient() bool
	PError() perror.Error
	String() string

	err()
}
