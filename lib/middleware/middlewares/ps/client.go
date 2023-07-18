package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	parameters map[string]string

	peer  *Server
	dirty bool

	middleware.Nil
}

func NewClient() *Client {
	return &Client{
		parameters: make(map[string]string),
	}
}

func (T *Client) SetServer(peer *Server) {
	T.dirty = true
	T.peer = peer
}

func (T *Client) updateParameter0(ctx middleware.Context, name, value string) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteParameterStatus(packet, name, value)
	err := ctx.Write(packet)
	if err != nil {
		return err
	}

	T.parameters[name] = value

	return nil
}

func (T *Client) updateParameter(ctx middleware.Context, name, value string) error {
	if T.parameters[name] == value {
		return nil
	}

	return T.updateParameter0(ctx, name, value)
}

func (T *Client) sync(ctx middleware.Context) error {
	if T.peer == nil || !T.dirty {
		return nil
	}
	T.dirty = false

	for name, value := range T.parameters {
		expected := T.peer.parameters[name]
		if value == expected {
			continue
		}
		err := T.updateParameter0(ctx, name, expected)
		if err != nil {
			return err
		}
	}

	for name, expected := range T.peer.parameters {
		err := T.updateParameter(ctx, name, expected)
		if err != nil {
			return err
		}
	}

	return nil
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
	return T.sync(ctx)
}

var _ middleware.Middleware = (*Client)(nil)
