package eqp

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/ring"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type HashedPortal struct {
	source string
	hash   uint64
}

type Server struct {
	preparedStatements map[string]uint64
	portals            map[string]HashedPortal

	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]
	pendingCloses             ring.Ring[Close]

	peer *Client

	middleware.Nil
}

func NewServer() *Server {
	return &Server{
		preparedStatements: make(map[string]uint64),
		portals:            make(map[string]HashedPortal),
	}
}

func (T *Server) SetClient(client *Client) {
	T.peer = client
}

func (T *Server) deletePreparedStatement(target string) {
	delete(T.preparedStatements, target)
}

func (T *Server) deletePortal(target string) {
	delete(T.portals, target)
}

func (T *Server) closePreparedStatement(ctx middleware.Context, target string) error {
	// no need to close unnamed prepared statement
	if target == "" {
		return nil
	}

	hash, ok := T.preparedStatements[target]
	if !ok {
		// already closed
		return nil
	}

	// send close packet
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteClose(packet, 'S', target)
	err := ctx.Write(packet)
	if err != nil {
		return err
	}

	// add it to pending
	delete(T.preparedStatements, target)
	T.pendingCloses.PushBack(Close{
		Which:  'S',
		Target: target,
		Hash:   hash,
	})
	return nil
}

func (T *Server) closePortal(ctx middleware.Context, target string) error {
	/*
		DON'T DO THIS!! Even though the unnamed portal doesn't need to be closed if the portal is ok, binding over an
		unrunnable portal will keep the portal in an unrunnable state.

		if target == "" {
			return nil
		}
	*/

	hash, ok := T.portals[target]
	if !ok {
		// already closed
		return nil
	}

	// send close packet
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteClose(packet, 'P', target)
	err := ctx.Write(packet)
	if err != nil {
		return err
	}

	// add it to pending
	delete(T.portals, target)
	T.pendingCloses.PushBack(Close{
		Which:  'P',
		Target: target,
		Source: hash.source,
		Hash:   hash.hash,
	})
	return nil
}

func (T *Server) bindPreparedStatement(
	ctx middleware.Context,
	target string,
	preparedStatement PreparedStatement,
) error {
	err := T.closePreparedStatement(ctx, target)
	if err != nil {
		return err
	}

	err = ctx.Write(preparedStatement.packet)
	if err != nil {
		return err
	}

	T.deletePreparedStatement(target)
	T.preparedStatements[target] = preparedStatement.hash
	T.pendingPreparedStatements.PushBack(target)
	return nil
}

func (T *Server) bindPortal(
	ctx middleware.Context,
	target string,
	portal Portal,
) error {
	// check if we already have it bound
	if old, ok := T.portals[target]; ok {
		if old.hash == portal.hash {
			return nil
		}
	}

	err := T.closePortal(ctx, target)
	if err != nil {
		return err
	}

	err = ctx.Write(portal.packet)
	if err != nil {
		return err
	}

	T.deletePortal(target)
	T.portals[target] = HashedPortal{
		source: portal.source,
		hash:   portal.hash,
	}
	T.pendingPortals.PushBack(target)
	return nil
}

func (T *Server) syncPreparedStatement(ctx middleware.Context, target string) error {
	expected, some := T.peer.preparedStatements[target]
	if !some {
		return T.closePreparedStatement(ctx, target)
	}

	// check if we already have it bound
	if old, ok := T.preparedStatements[target]; ok {
		if old == expected.hash {
			return nil
		}
	}

	// clear all portals that use this prepared statement
	for name, portal := range T.portals {
		if portal.source == target {
			err := T.closePortal(ctx, name)
			if err != nil {
				return err
			}
		}
	}

	return T.bindPreparedStatement(ctx, target, expected)
}

func (T *Server) syncPortal(ctx middleware.Context, target string) error {
	expected, some := T.peer.portals[target]
	if !some {
		return T.closePortal(ctx, target)
	}

	err := T.syncPreparedStatement(ctx, expected.source)
	if err != nil {
		return err
	}

	// check if we already have it bound
	if old, ok := T.portals[target]; ok {
		if old.hash == expected.hash {
			return nil
		}
	}

	return T.bindPortal(ctx, target, expected)
}

func (T *Server) Write(ctx middleware.Context, packet *zap.Packet) error {
	switch packet.ReadType() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		T.deletePreparedStatement("")
		T.deletePortal("")
	case packets.Parse, packets.Bind, packets.Close:
		// should've been caught by eqp.Client
		panic("unreachable")
	case packets.Describe:
		// ensure target exists
		which, target, ok := packets.ReadDescribe(packet.Read())
		if !ok {
			// should've been caught by eqp.Client
			panic("unreachable")
		}
		switch which {
		case 'S':
			// sync prepared statement
			err := T.syncPreparedStatement(ctx, target)
			if err != nil {
				return err
			}
		case 'P':
			// sync portal
			err := T.syncPortal(ctx, target)
			if err != nil {
				return err
			}
		default:
			panic("unknown describe target")
		}
	case packets.Execute:
		target, _, ok := packets.ReadExecute(packet.Read())
		if !ok {
			// should've been caught by eqp.Client
			panic("unreachable")
		}
		// sync portal
		err := T.syncPortal(ctx, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (T *Server) Read(ctx middleware.Context, packet *zap.Packet) error {
	read := packet.Read()
	switch read.ReadType() {
	case packets.ParseComplete:
		ctx.Cancel()

		T.pendingPreparedStatements.PopFront()
	case packets.BindComplete:
		ctx.Cancel()

		T.pendingPortals.PopFront()
	case packets.CloseComplete:
		ctx.Cancel()

		T.pendingCloses.PopFront()
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(packet.Read())
		if !ok {
			return errors.New("bad packet format")
		}
		if state == 'I' {
			// clobber all portals
			for name := range T.portals {
				T.deletePortal(name)
			}
		}
		// all pending failed
		for pending, ok := T.pendingPreparedStatements.PopBack(); ok; pending, ok = T.pendingPreparedStatements.PopBack() {
			T.deletePreparedStatement(pending)
		}
		for pending, ok := T.pendingPortals.PopBack(); ok; pending, ok = T.pendingPortals.PopBack() {
			T.deletePortal(pending)
		}
		for pending, ok := T.pendingCloses.PopBack(); ok; pending, ok = T.pendingCloses.PopBack() {
			switch pending.Which {
			case 'S': // prepared statement
				T.deletePreparedStatement(pending.Target)
				T.preparedStatements[pending.Target] = pending.Hash
			case 'P': // portal
				T.deletePortal(pending.Target)
				T.portals[pending.Target] = HashedPortal{
					hash:   pending.Hash,
					source: pending.Source,
				}
			default:
				panic("unreachable")
			}
		}
	}
	return nil
}

func (T *Server) Done() {
	for name := range T.preparedStatements {
		T.deletePreparedStatement(name)
	}
	for name := range T.portals {
		T.deletePortal(name)
	}
	for _, ok := T.pendingCloses.PopBack(); ok; _, ok = T.pendingCloses.PopBack() {
	}
}

var _ middleware.Middleware = (*Server)(nil)
