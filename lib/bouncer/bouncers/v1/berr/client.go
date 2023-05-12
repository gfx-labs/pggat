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

func (Client) err() {}

var _ Error = Client{}
