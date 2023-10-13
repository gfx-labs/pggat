package ps

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func sync(tracking []strutil.CIString, client *fed.Conn, c *Client, server *fed.Conn, s *Server, name strutil.CIString) error {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	if value == expected {
		if !c.synced {
			ps := packets.ParameterStatus{
				Key:   name.String(),
				Value: expected,
			}
			if err := client.WritePacket(&ps); err != nil {
				return err
			}
		}
		return nil
	}

	var doSet bool

	if hasValue && slices.Contains(tracking, name) {
		var err error
		if err, _ = backends.SetParameter(server, nil, name, value); err != nil {
			return err
		}
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
		if err := client.WritePacket(&ps); err != nil {
			return err
		}
	}

	return nil
}

func Sync(tracking []strutil.CIString, client, server *fed.Conn) (clientErr, serverErr error) {
	c, ok := fed.LookupMiddleware[*Client](client)
	if !ok {
		panic("middleware not found")
	}
	s, ok := fed.LookupMiddleware[*Server](server)
	if !ok {
		panic("middleware not found")
	}

	for name := range c.parameters {
		if serverErr = sync(tracking, client, c, server, s, name); serverErr != nil {
			return
		}
	}

	for name := range s.parameters {
		if _, ok = c.parameters[name]; ok {
			continue
		}
		if serverErr = sync(tracking, client, c, server, s, name); serverErr != nil {
			return
		}
	}

	c.synced = true

	return
}
