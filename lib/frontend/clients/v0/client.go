package clients

import (
	"net"

	"pggat2/lib/frontend"
)

type Client struct {
	conn net.Conn
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

var _ frontend.Client = (*Client)(nil)
