package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Server struct {
	parameters map[string]string

	middleware.Nil
}

func NewServer(parameters map[string]string) *Server {
	return &Server{
		parameters: parameters,
	}
}

func (T *Server) syncParameter(pkts *zap.Packets, ps *Client, name, expected string) {
	packet := zap.NewPacket()
	packets.WriteParameterStatus(packet, name, expected)
	pkts.Append(packet)

	ps.parameters[name] = expected
}

func (T *Server) Sync(client zap.ReadWriter, ps *Client) error {
	pkts := zap.NewPackets()
	defer pkts.Done()

	for name, value := range ps.parameters {
		expected := T.parameters[name]
		if value == expected {
			continue
		}

		T.syncParameter(pkts, ps, name, expected)
	}

	for name, expected := range T.parameters {
		if T.parameters[name] == expected {
			continue
		}

		T.syncParameter(pkts, ps, name, expected)
	}

	return client.WriteV(pkts)
}

func (T *Server) Read(_ middleware.Context, in *zap.Packet) error {
	read := in.Read()
	switch read.ReadType() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(&read)
		if !ok {
			return errors.New("bad packet format")
		}
		T.parameters[key] = value
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
