package eqp

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type Client struct {
	state State
}

func NewClient() *Client {
	return new(Client)
}

func (T *Client) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	T.state.C2S(packet)
	return packet, nil
}

func (T *Client) WritePacket(packet fed.Packet) (fed.Packet, error) {
	T.state.S2C(packet)
	return packet, nil
}

var _ fed.Middleware = (*Client)(nil)
