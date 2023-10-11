package eqp

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Sync(c *Client, server *fed.Conn, s *Server) error {
	var needsBackendSync bool

	// close all portals on server
	// we close all because there won't be any for the normal case anyway, and it's hard to tell
	// if a portal is accurate because the underlying prepared statement could have changed.
	if len(s.state.portals) > 0 {
		needsBackendSync = true
	}

	for name := range s.state.portals {
		p := packets.Close{
			Which: 'P',
			Name:  name,
		}
		if err := server.WritePacket(&p); err != nil {
			return err
		}
	}

	// close all prepared statements that don't match client
	for name, preparedStatement := range s.state.preparedStatements {
		if clientPreparedStatement, ok := c.state.preparedStatements[name]; ok {
			// TODO(garet) do not overwrite prepared statements that match
			_ = preparedStatement
			_ = clientPreparedStatement

			if name == "" {
				// will be overwritten
				continue
			}
		}

		p := packets.Close{
			Which: 'S',
			Name:  name,
		}
		if err := server.WritePacket(&p); err != nil {
			return err
		}

		needsBackendSync = true
	}

	// parse all prepared statements that aren't on server
	for name, preparedStatement := range c.state.preparedStatements {
		if serverPreparedStatement, ok := s.state.preparedStatements[name]; ok {
			// TODO(garet) do not overwrite prepared statements that match
			_ = preparedStatement
			_ = serverPreparedStatement
		}

		if err := server.WritePacket(preparedStatement); err != nil {
			return err
		}

		needsBackendSync = true
	}

	// bind all portals
	if len(c.state.portals) > 0 {
		needsBackendSync = true
	}

	for _, portal := range c.state.portals {
		if err := server.WritePacket(portal); err != nil {
			return err
		}
	}

	if needsBackendSync {
		var err error
		err, _ = backends.Sync(server, nil)
		return err
	}

	return nil
}
