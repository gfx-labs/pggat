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

func (T *Client) Write(ctx middleware.Context, packet zap.Packet) error {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			return errors.New("bad packet format")
		}
		ikey := strutil.MakeCIString(ps.Key)
		if T.parameters[ikey] == ps.Value {
			// already set
			ctx.Cancel()
			break
		}
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = ps.Value
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
