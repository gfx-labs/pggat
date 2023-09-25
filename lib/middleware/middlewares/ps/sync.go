package ps

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func sync(tracking []strutil.CIString, client fed.ReadWriter, c *Client, server fed.ReadWriter, s *Server, name strutil.CIString) error {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	var packet fed.Packet

	if value == expected {
		if !c.synced {
			ps := packets.ParameterStatus{
				Key:   name.String(),
				Value: expected,
			}
			packet = ps.IntoPacket(packet)
			if err := client.WritePacket(packet); err != nil {
				return err
			}
		}
		return nil
	}

	var doSet bool

	if hasValue && slices.Contains(tracking, name) {
		ctx := backends.Context{
			Packet: packet,
			Server: server,
		}
		if err := backends.SetParameter(&ctx, name, value); err != nil {
			return err
		}
		packet = ctx.Packet
		if s.parameters == nil {
			s.parameters = make(map[strutil.CIString]string)
		}
		s.parameters[name] = value

		doSet = true
	} else if hasExpected {
		doSet = true
	}

	if doSet {
		ps := packets.ParameterStatus{
			Key:   name.String(),
			Value: expected,
		}
		packet = ps.IntoPacket(packet)
		if err := client.WritePacket(packet); err != nil {
			return err
		}
	}

	return nil
}

func Sync(tracking []strutil.CIString, client fed.ReadWriter, c *Client, server fed.ReadWriter, s *Server) (clientErr, serverErr error) {
	for name := range c.parameters {
		if serverErr = sync(tracking, client, c, server, s, name); serverErr != nil {
			return
		}
	}

	for name := range s.parameters {
		if _, ok := c.parameters[name]; ok {
			continue
		}
		if serverErr = sync(tracking, client, c, server, s, name); serverErr != nil {
			return
		}
	}

	c.synced = true

	return
}
