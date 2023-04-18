package client

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
	"gfx.cafe/gfx/pggat/lib/gat2/source"
)

type Client struct {
	out    chan request.Request
	closed chan struct{}
}

func NewClient() *Client {
	return &Client{
		out:    make(chan request.Request),
		closed: make(chan struct{}),
	}
}

func (T *Client) Out() <-chan request.Request {
	return T.out
}

func (T *Client) Closed() <-chan struct{} {
	return T.closed
}

var _ source.Source = (*Client)(nil)
