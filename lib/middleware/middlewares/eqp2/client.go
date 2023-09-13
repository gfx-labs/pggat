package eqp2

import (
	"pggat/lib/fed"
	"pggat/lib/middleware"
)

type Client struct {
}

func (T *Client) Read(ctx middleware.Context, packet fed.Packet) error {
	// TODO implement me
	panic("implement me")
}

func (T *Client) Write(ctx middleware.Context, packet fed.Packet) error {
	// TODO implement me
	panic("implement me")
}

var _ middleware.Middleware = (*Client)(nil)
