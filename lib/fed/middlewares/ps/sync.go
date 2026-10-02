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

	if slices.Contains(tracking, name) {
		desired, hasDesired := value, hasValue
		if !hasDesired {
			desired, hasDesired = s.initialParameters[name]
		}
		if !hasDesired {
			// Older servers may not report this parameter at startup or after SET.
			if hasExpected {
				if serverErr, _ = backends.ResetParameter(ctx, server, nil, name); serverErr != nil {
					return
				}
				delete(s.parameters, name)
			}
			return
		}
		if !hasExpected || desired != expected {
			if serverErr, _ = backends.SetParameter(ctx, server, nil, name, desired); serverErr != nil {
				return
			}
			if s.parameters == nil {
				s.parameters = make(map[strutil.CIString]string)
			}
			// Keep canonical values reported by the server. Unreported parameters
			// still need bookkeeping for startup values and subsequent resets.
			if _, reported := s.initialParameters[name]; !reported {
				s.parameters[name] = desired
			}
			expected, hasExpected = s.parameters[name]
		}
	}

	if client != nil && hasExpected && (!c.synced || !hasValue || value != expected) {
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
