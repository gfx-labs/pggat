package berr

import "pggat2/lib/perror"

type Client struct {
	Error perror.Error
}

func (Client) IsServer() bool {
	return false
}

func (Client) IsClient() bool {
	return true
}

func (T Client) PError() perror.Error {
	return T.Error
}

func (T Client) String() string {
	return T.Error.Message()
}

func (Client) err() {}

var _ Error = Client{}
