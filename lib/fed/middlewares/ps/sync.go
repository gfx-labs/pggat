package ps

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func sync(ctx context.Context, tracking []strutil.CIString, client *fed.Conn, c *Client, server *fed.Conn, s *Server, name strutil.CIString) (clientErr, serverErr error) {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	if value == expected {
		if client != nil && !c.synced {
			ps := packets.ParameterStatus{
				Key:   name.String(),
				Value: expected,
			}
			clientErr = client.WritePacket(ctx, &ps)
		}
		return
	}

	var doSet bool

	if hasValue && slices.Contains(tracking, name) {
		if serverErr, _ = backends.SetParameter(ctx, server, nil, name, value); serverErr != nil {
			return
		}
		if s.parameters == nil {
			s.parameters = make(map[strutil.CIString]string)
		}
		s.parameters[name] = value
		expected = value

		if !c.synced {
			doSet = true
		}
	} else if hasExpected {
		doSet = true
	}

	if client != nil && doSet {
		ps := packets.ParameterStatus{
			Key:   name.String(),
			Value: expected,
		}
		if clientErr = client.WritePacket(ctx, &ps); clientErr != nil {
			return
		}
	}

	return
}

func SyncMiddleware(ctx context.Context, tracking []strutil.CIString, c *Client, server *fed.Conn) error {
	s, ok := fed.LookupMiddleware[*Server](server)
	if !ok {
		panic("middleware not found")
	}

	for name := range c.parameters {
		if _, err := sync(ctx, tracking, nil, c, server, s, name); err != nil {
			return err
		}
	}

	for name := range s.parameters {
		if _, ok = c.parameters[name]; ok {
			continue
		}
		if _, err := sync(ctx, tracking, nil, c, server, s, name); err != nil {
			return err
		}
	}

	return nil
}

func Sync(ctx context.Context, tracking []strutil.CIString, client, server *fed.Conn) (clientErr, serverErr error) {
	c, ok := fed.LookupMiddleware[*Client](client)
	if !ok {
		panic("middleware not found")
	}
	s, ok := fed.LookupMiddleware[*Server](server)
	if !ok {
		panic("middleware not found")
	}

	for name := range c.parameters {
		if clientErr, serverErr = sync(ctx, tracking, client, c, server, s, name); clientErr != nil || serverErr != nil {
			return
		}
	}

	for name := range s.parameters {
		if _, ok = c.parameters[name]; ok {
			continue
		}
		if clientErr, serverErr = sync(ctx, tracking, client, c, server, s, name); clientErr != nil || serverErr != nil {
			return
		}
	}

	c.synced = true

	return
}
