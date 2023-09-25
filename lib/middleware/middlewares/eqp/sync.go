package eqp

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Sync(c *Client, server fed.ReadWriter, s *Server) error {
	var needsBackendSync bool

	// close all portals on server
	// we close all because there won't be any for the normal case anyway, and it's hard to tell
	// if a portal is accurate because the underlying prepared statement could have changed.
	if len(s.state.portals) > 0 {
		needsBackendSync = true
	}

	var packet fed.Packet

	for name := range s.state.portals {
		p := packets.Close{
			Which:  'P',
			Target: name,
		}
		packet = p.IntoPacket(packet)
		if err := server.WritePacket(packet); err != nil {
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
		packet = p.IntoPacket(packet)
		if err := server.WritePacket(packet); err != nil {
			return err
		}

		needsBackendSync = true
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

		needsBackendSync = true
	}

	// bind all portals
	if len(c.state.portals) > 0 {
		needsBackendSync = true
	}

	for _, portal := range c.state.portals {
		if err := server.WritePacket(portal.Packet); err != nil {
			return err
		}
	}

	if needsBackendSync {
		ctx := backends.Context{
			Packet: packet,
			Server: server,
		}
		_, err := backends.Sync(&ctx)
		return err
	}

	return nil
}
