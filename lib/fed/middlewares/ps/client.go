package ps

import (
	"errors"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
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

func (T *Client) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

func (T *Client) WritePacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			return packet, errors.New("bad packet format i")
		}
		ikey := strutil.MakeCIString(ps.Key)
		if T.synced && T.parameters[ikey] == ps.Value {
			// already set
			return packet[:0], nil
		}
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = ps.Value
	}
	return packet, nil
}

var _ fed.Middleware = (*Client)(nil)
