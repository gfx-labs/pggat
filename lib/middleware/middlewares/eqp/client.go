package eqp

import (
	"pggat/lib/fed"
	"pggat/lib/middleware"
)

type Client struct {
	state State
}

func NewClient() *Client {
	return new(Client)
}

func (T *Client) Read(_ middleware.Context, packet fed.Packet) error {
	T.state.C2S(packet)
	return nil
}

func (T *Client) Write(_ middleware.Context, packet fed.Packet) error {
	T.state.S2C(packet)
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
