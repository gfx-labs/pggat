package eqp

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

func preparedStatementsEqual(a, b *packets.Parse) bool {
	if a.Query != b.Query {
		return false
	}

	if !slices.Equal(a.ParameterDataTypes, b.ParameterDataTypes) {
		return false
	}

	return true
}

func SyncMiddleware(c *Client, server *fed.Conn) error {
	s, ok := fed.LookupMiddleware[*Server](server)
	if !ok {
		panic("middleware not found")
	}

	var needsBackendSync bool

	// close all portals on server
	// we close all because there won't be any for the normal case anyway, and it's hard to tell
	// if a portal is accurate because the underlying prepared statement could have changed.
	for name := range s.state.portals {
		p := packets.Close{
			Which: 'P',
			Name:  name,
		}
		if err := server.WritePacket(&p); err != nil {
			return err
		}

		needsBackendSync = true
	}

	// close all prepared statements that don't match client
	for name, preparedStatement := range s.state.preparedStatements {
		if clientPreparedStatement, ok := c.state.preparedStatements[name]; ok {
			if preparedStatementsEqual(preparedStatement, clientPreparedStatement) {
				continue
			}

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
			if preparedStatementsEqual(preparedStatement, serverPreparedStatement) {
				continue
			}
		}

		if err := server.WritePacket(preparedStatement); err != nil {
			return err
		}

		needsBackendSync = true
	}

	// bind all portals
	for _, portal := range c.state.portals {
		if err := server.WritePacket(portal); err != nil {
			return err
		}

		needsBackendSync = true
	}

	if needsBackendSync {
		var err error
		err, _ = backends.Sync(server, nil)
		return err
	}

	return nil
}

func Sync(client, server *fed.Conn) error {
	c, ok := fed.LookupMiddleware[*Client](client)
	if !ok {
		panic("middleware not found")
	}

	return SyncMiddleware(c, server)
}
