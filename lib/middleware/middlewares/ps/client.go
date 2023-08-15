package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	parameters map[strutil.CIString]string

	middleware.Nil
}

func NewClient() *Client {
	return &Client{
		parameters: make(map[strutil.CIString]string),
	}
}

func (T *Client) Send(ctx middleware.Context, packet *zap.Packet) error {
	switch packet.ReadType() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(packet.Read())
		if !ok {
			return errors.New("bad packet format")
		}
		ikey := strutil.MakeCIString(key)
		if T.parameters[ikey] == value {
			// already set
			ctx.Cancel()
			break
		}
		T.parameters[ikey] = value
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
