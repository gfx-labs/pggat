package eqp

import (
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
)

func Sync(c *Client, server fed.ReadWriter, s *Server) error {
	// close all portals on server
	// we close all because there won't be any for the normal case anyway, and it's hard to tell
	// if a portal is accurate because the underlying prepared statement could have changed.
	for name := range s.state.portals {
		p := packets.Close{
			Which:  'P',
			Target: name,
		}
		if err := server.WritePacket(p.IntoPacket()); err != nil {
			return err
		}
	}

	// close all prepared statements that don't match client
	for name, preparedStatement := range s.state.preparedStatements {
		if clientPreparedStatement, ok := c.state.preparedStatements[name]; ok {
			if preparedStatement.Hash == clientPreparedStatement.Hash {
				// the same
				continue
			}

			if name == "" {
				// will be overwritten
				continue
			}
		}

		p := packets.Close{
			Which:  'S',
			Target: name,
		}
		if err := server.WritePacket(p.IntoPacket()); err != nil {
			return err
		}
	}

	// parse all prepared statements that aren't on server
	for name, preparedStatement := range c.state.preparedStatements {
		if serverPreparedStatement, ok := s.state.preparedStatements[name]; ok {
			if preparedStatement.Hash == serverPreparedStatement.Hash {
				// the same
				continue
			}
		}

		if err := server.WritePacket(preparedStatement.Packet); err != nil {
			return err
		}
	}

	// bind all portals
	for _, portal := range c.state.portals {
		if err := server.WritePacket(portal.Packet); err != nil {
			return err
		}
	}

	_, err := backends.Sync(new(backends.Context), server)
	return err
}
