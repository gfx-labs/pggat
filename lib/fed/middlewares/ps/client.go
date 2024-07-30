package ps

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/maps"
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

func (T *Client) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Client) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

func (T *Client) WritePacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var p packets.ParameterStatus
		err := fed.ToConcrete(&p, packet)
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
		return &p, nil
	default:
		return packet, nil
	}
}

func (T *Client) PostWrite() (fed.Packet, error) {
	return nil, nil
}

func (T *Client) Set(other *Client) {
	T.synced = other.synced

	maps.Clear(T.parameters)
	if T.parameters == nil {
		T.parameters = make(map[strutil.CIString]string)
	}
	for k, v := range other.parameters {
		T.parameters[k] = v
	}
}

var _ fed.Middleware = (*Client)(nil)
