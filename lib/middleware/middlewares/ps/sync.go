package ps

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func sync(tracking []strutil.CIString, clientPackets *zap.Packets, c *Client, server zap.ReadWriter, s *Server, name strutil.CIString) error {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	if value == expected {
		if !c.synced {
			pkt := zap.NewPacket()
			packets.WriteParameterStatus(pkt, name.String(), expected)
			clientPackets.Append(pkt)
		}
		return nil
	}

	if slices.Contains(tracking, name) {
		if hasValue {
			if err := backends.SetParameter(&backends.Context{}, server, name, value); err != nil {
				return err
			}
			if s.parameters == nil {
				s.parameters = make(map[strutil.CIString]string)
			}
			s.parameters[name] = value
		} else {
			if err := backends.ResetParameter(&backends.Context{}, server, name); err != nil {
				return err
			}
			delete(s.parameters, name)
		}
	} else if hasExpected {
		pkt := zap.NewPacket()
		packets.WriteParameterStatus(pkt, name.String(), expected)
		clientPackets.Append(pkt)
	}

	return nil
}

func Sync(tracking []strutil.CIString, client zap.ReadWriter, c *Client, server zap.ReadWriter, s *Server) (clientErr, serverErr error) {
	pkts := zap.NewPackets()
	defer pkts.Done()

	for name := range c.parameters {
		if serverErr = sync(tracking, pkts, c, server, s, name); serverErr != nil {
			return
		}
	}

	for name := range s.parameters {
		if _, ok := c.parameters[name]; ok {
			continue
		}
		if serverErr = sync(tracking, pkts, c, server, s, name); serverErr != nil {
			return
		}
	}

	c.synced = true

	if clientErr = client.WriteV(pkts); clientErr != nil {
		return
	}

	return
}
