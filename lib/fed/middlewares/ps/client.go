package ps

import (
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
		p, err := fed.ToConcrete[*packets.ParameterStatus](packet)
		if err != nil {
			return nil, err
		}
		ikey := strutil.MakeCIString(p.Key)
		if T.synced && T.parameters[ikey] == p.Value {
			// already set
			return nil, nil
		}
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = p.Value
		return p, nil
	default:
		return packet, nil
	}
}

var _ fed.Middleware = (*Client)(nil)
