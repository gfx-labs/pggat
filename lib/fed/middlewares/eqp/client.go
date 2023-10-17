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

func (T *Client) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Client) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	return T.state.C2S(packet)
}

func (T *Client) WritePacket(packet fed.Packet) (fed.Packet, error) {
	return T.state.S2C(packet)
}

func (T *Client) PostWrite() (fed.Packet, error) {
	return nil, nil
}

var _ fed.Middleware = (*Client)(nil)
