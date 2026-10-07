package eqp

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
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

// SyncMiddleware makes the server's prepared statements and portals match the client's.
// Prepared statements are parsed lazily, just before the client next uses them, so a statement
// that no longer parses fails only that request, with its own error.
func SyncMiddleware(ctx context.Context, c *Client, server *fed.Conn) error {
	s, ok := fed.LookupMiddleware[*Server](server)
	if !ok {
		panic("middleware not found")
	}

	clear(s.lazy)
	s.deferred = nil

	var needsBackendSync bool

	// close all portals on server
	// we close all because there won't be any for the normal case anyway, and it's hard to tell
	// if a portal is accurate because the underlying prepared statement could have changed.
	for name := range s.state.portals {
		p := packets.Close{
			Which: 'P',
			Name:  name,
		}
		if err := server.WritePacket(ctx, &p); err != nil {
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
		if err := server.WritePacket(ctx, &p); err != nil {
			return err
		}

		needsBackendSync = true
	}

	// defer every prepared statement that isn't on server
	for name, preparedStatement := range c.state.preparedStatements {
		if serverPreparedStatement, ok := s.state.preparedStatements[name]; ok {
			if preparedStatementsEqual(preparedStatement, serverPreparedStatement) {
				continue
			}
		}

		if s.lazy == nil {
			s.lazy = make(map[string]*packets.Parse)
		}
		s.lazy[name] = preparedStatement
	}

	// bind all portals. Binding injects the Parse of their prepared statements.
	for _, portal := range c.state.portals {
		if err := server.WritePacket(ctx, portal); err != nil {
			return err
		}

		needsBackendSync = true
	}

	if !needsBackendSync {
		return nil
	}

	if err := server.WritePacket(ctx, &packets.Sync{}); err != nil {
		return err
	}

	// Close does not fail, so an error belongs to a client portal or the Parse injected for it.
	// The portal cannot be recreated, so report it.
	var serverErr error
	s.syncing = true
	defer func() { s.syncing = false }()
	for {
		packet, err := server.ReadPacket(ctx, true)
		if err != nil {
			return err
		}
		switch packet.Type() {
		case packets.TypeBindComplete,
			packets.TypeCloseComplete,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
		case packets.TypeMarkiplierResponse:
			if serverErr == nil {
				var p packets.MarkiplierResponse
				if err = fed.ToConcrete(&p, packet); err != nil {
					return err
				}
				serverErr = perror.FromPacket(&p)
			}
		case packets.TypeReadyForQuery:
			return serverErr
		default:
			return backends.ErrUnexpectedPacket(packet.Type())
		}
	}
}

func Sync(ctx context.Context, client, server *fed.Conn) error {
	c, ok := fed.LookupMiddleware[*Client](client)
	if !ok {
		panic("middleware not found")
	}

	return SyncMiddleware(ctx, c, server)
}
