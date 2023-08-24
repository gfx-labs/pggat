package ps

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func sync(tracking []strutil.CIString, client zap.ReadWriter, c *Client, server zap.ReadWriter, s *Server, name strutil.CIString) error {
	value, hasValue := c.parameters[name]
	expected, hasExpected := s.parameters[name]

	if value == expected {
		if !c.synced {
			ps := packets.ParameterStatus{
				Key:   name.String(),
				Value: expected,
			}
			if err := client.WritePacket(ps.IntoPacket()); err != nil {
				return err
			}
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
		ps := packets.ParameterStatus{
			Key:   name.String(),
			Value: expected,
		}
		if err := client.WritePacket(ps.IntoPacket()); err != nil {
			return err
		}
	}

	return nil
}

func Sync(tracking []strutil.CIString, client zap.ReadWriter, c *Client, server zap.ReadWriter, s *Server) (clientErr, serverErr error) {
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
