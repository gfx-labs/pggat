package ps

import (
	"errors"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Client struct {
	synced     bool
	parameters map[strutil.CIString]string
}

func NewClient(parameters map[strutil.CIString]string) *Client {
	return &Client{
		parameters: parameters,
	}
}

func (T *Client) Read(_ middleware.Context, _ fed.Packet) error {
	return nil
}

func (T *Client) Write(ctx middleware.Context, packet fed.Packet) error {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			return errors.New("bad packet format i")
		}
		ikey := strutil.MakeCIString(ps.Key)
		if T.synced && T.parameters[ikey] == ps.Value {
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
