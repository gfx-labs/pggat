package ps

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func sync(tracking []strutil.CIString, clientPackets *zap.Packets, c *Client, server zap.ReadWriter, s *Server, name strutil.CIString) {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	if value == expected {
		if !c.synced {
			pkt := zap.NewPacket()
			packets.WriteParameterStatus(pkt, name.String(), expected)
			clientPackets.Append(pkt)
		}
		return
	}

	if hasValue && slices.Contains(tracking, name) {
		if err := backends.SetParameter(&backends.Context{}, server, name, value); err != nil {
			panic(err) // TODO(garet)
		}
		if s.parameters == nil {
			s.parameters = make(map[strutil.CIString]string)
		}
	} else if hasExpected {
		pkt := zap.NewPacket()
		packets.WriteParameterStatus(pkt, name.String(), expected)
		clientPackets.Append(pkt)
		if c.parameters == nil {
			c.parameters = make(map[strutil.CIString]string)
		}
	}
}

func Sync(tracking []strutil.CIString, client zap.ReadWriter, c *Client, server zap.ReadWriter, s *Server) {
	pkts := zap.NewPackets()
	defer pkts.Done()

	for name := range c.parameters {
		sync(tracking, pkts, c, server, s, name)
	}

	for name := range s.parameters {
		if _, ok := c.parameters[name]; ok {
			continue
		}
		sync(tracking, pkts, c, server, s, name)
	}

	c.synced = true

	if err := client.WriteV(pkts); err != nil {
		panic(err) // TODO(garet)
	}
}
