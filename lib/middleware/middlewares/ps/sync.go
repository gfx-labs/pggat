package ps

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func sync(tracking []strutil.CIString, clientPackets *zap.Packets, c *Client, server zap.ReadWriter, s *Server, name strutil.CIString) {
	value := c.parameters[name]
	expected := s.parameters[name]

	if value == expected {
		// TODO(garet) this will send twice if both server and client have it
		if !c.synced {
			pkt := zap.NewPacket()
			packets.WriteParameterStatus(pkt, name.String(), expected)
			clientPackets.Append(pkt)
		}
		return
	}

	if slices.Contains(tracking, name) {
		if err := backends.QueryString(&backends.Context{}, server, `SET `+strutil.Escape(name.String(), `"`)+` = `+strutil.Escape(value, `'`)); err != nil {
			panic(err) // TODO(garet)
		}
		if s.parameters == nil {
			s.parameters = make(map[strutil.CIString]string)
		}
		s.parameters[name] = value
	} else {
		pkt := zap.NewPacket()
		packets.WriteParameterStatus(pkt, name.String(), expected)
		clientPackets.Append(pkt)
		if c.parameters == nil {
			c.parameters = make(map[strutil.CIString]string)
		}
		c.parameters[name] = value
	}
}

func Sync(tracking []strutil.CIString, client zap.ReadWriter, c *Client, server zap.ReadWriter, s *Server) {
	pkts := zap.NewPackets()
	defer pkts.Done()

	for name := range c.parameters {
		sync(tracking, pkts, c, server, s, name)
	}

	for name := range s.parameters {
		sync(tracking, pkts, c, server, s, name)
	}

	c.synced = true

	if err := client.WriteV(pkts); err != nil {
		panic(err) // TODO(garet)
	}
}
