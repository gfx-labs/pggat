package eqp

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type Client struct {
	state State
}

func NewClient() *Client {
	return new(Client)
}

func (T *Client) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Client) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	return T.state.C2S(packet)
}

func (T *Client) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	return T.state.S2C(packet)
}

func (T *Client) PostWrite(ctx context.Context) (fed.Packet, error) {
	return nil, nil
}

func (T *Client) Set(ctx context.Context, other *Client) {
	T.state.Set(&other.state)
}

var _ fed.Middleware = (*Client)(nil)
