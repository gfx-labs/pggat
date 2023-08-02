package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	parameters map[string]string

	middleware.Nil
}

func NewClient() *Client {
	return &Client{
		parameters: make(map[string]string),
	}
}

func (T *Client) Send(ctx middleware.Context, packet *zap.Packet) error {
	read := packet.Read()
	switch read.ReadType() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(&read)
		if !ok {
			return errors.New("bad packet format")
		}
		if T.parameters[key] == value {
			// already set
			ctx.Cancel()
			break
		}
		T.parameters[key] = value
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
