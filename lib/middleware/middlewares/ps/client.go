package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	synced     bool
	parameters map[strutil.CIString]string

	middleware.Nil
}

func NewClient(parameters map[strutil.CIString]string) *Client {
	return &Client{
		parameters: parameters,
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
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = value
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
