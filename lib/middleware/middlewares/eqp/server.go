package eqp

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/ring"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Server struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal

	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]
	pendingCloses             ring.Ring[Close]

	buf zap.Buf

	peer *Client
}

func MakeServer() Server {
	return Server{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
	}
}

func (T *Server) SetClient(client *Client) {
	T.peer = client
}

func (T *Server) deletePreparedStatement(target string) {
	v, ok := T.preparedStatements[target]
	if !ok {
		return
	}
	v.Done()
	delete(T.preparedStatements, target)
}

func (T *Server) deletePortal(target string) {
	v, ok := T.portals[target]
	if !ok {
		return
	}
	v.Done()
	delete(T.portals, target)
}

func (T *Server) closePreparedStatement(ctx middleware.Context, target string) error {
	// no need to close unnamed prepared statement
	if target == "" {
		return nil
	}

	preparedStatement, ok := T.preparedStatements[target]
	if !ok {
		// already closed
		return nil
	}

	// send close packet
	out := T.buf.Write()
	packets.WriteClose(out, 'S', target)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	// add it to pending
	delete(T.preparedStatements, target)
	T.pendingCloses.PushBack(ClosePreparedStatement{
		target:            target,
		preparedStatement: preparedStatement,
	})
	return nil
}

func (T *Server) closePortal(ctx middleware.Context, target string) error {
	// no need to close unnamed portal
	if target == "" {
		return nil
	}

	portal, ok := T.portals[target]
	if !ok {
		// already closed
		return nil
	}

	// send close packet
	out := T.buf.Write()
	packets.WriteClose(out, 'P', target)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	// add it to pending
	delete(T.portals, target)
	T.pendingCloses.PushBack(ClosePortal{
		target: target,
		portal: portal,
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

	old := T.buf.Swap(preparedStatement.raw)
	err = ctx.Send(T.buf.Out())
	T.buf.Swap(old)
	if err != nil {
		return err
	}

	T.deletePreparedStatement(target)
	T.preparedStatements[target] = preparedStatement.Clone()
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
		if old.Equal(&portal) {
			return nil
		}
	}

	err := T.closePortal(ctx, target)
	if err != nil {
		return err
	}

	old := T.buf.Swap(portal.raw)
	err = ctx.Send(T.buf.Out())
	T.buf.Swap(old)
	if err != nil {
		return err
	}

	T.deletePortal(target)
	T.portals[target] = portal.Clone()
	T.pendingPortals.PushBack(target)
	return nil
}

func (T *Server) syncPreparedStatement(ctx middleware.Context, target string) error {
	expected := T.peer.preparedStatements[target]

	// check if we already have it bound
	if old, ok := T.preparedStatements[target]; ok {
		if old.Equal(&expected) {
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
	expected := T.peer.portals[target]

	err := T.syncPreparedStatement(ctx, expected.source)
	if err != nil {
		return err
	}

	// check if we already have it bound
	if old, ok := T.portals[target]; ok {
		if old.Equal(&expected) {
			return nil
		}
	}

	return T.bindPortal(ctx, target, expected)
}

func (T *Server) Send(ctx middleware.Context, out zap.Out) error {
	in := zap.OutToIn(out)
	switch in.Type() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		T.deletePreparedStatement("")
		T.deletePortal("")
	case packets.Parse, packets.Bind, packets.Close:
		// should've been caught by eqp.Client
		panic("unreachable")
	case packets.Describe:
		// ensure target exists
		which, target, ok := packets.ReadDescribe(in)
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
		target, _, ok := packets.ReadExecute(in)
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

func (T *Server) Read(ctx middleware.Context, in zap.In) error {
	switch in.Type() {
	case packets.ParseComplete:
		ctx.Cancel()

		T.pendingPreparedStatements.PopFront()
	case packets.BindComplete:
		ctx.Cancel()

		T.pendingPortals.PopFront()
	case packets.CloseComplete:
		ctx.Cancel()

		if c, ok := T.pendingCloses.PopFront(); ok {
			c.Done()
		}
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(in)
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
			switch p := pending.(type) {
			case ClosePreparedStatement:
				T.deletePreparedStatement(p.target)
				T.preparedStatements[p.target] = p.preparedStatement
			case ClosePortal:
				T.deletePortal(p.target)
				T.portals[p.target] = p.portal
			default:
				panic("unreachable")
			}
		}
	}
	return nil
}

func (T *Server) Done() {
	T.buf.Done()
	for name := range T.preparedStatements {
		T.deletePreparedStatement(name)
	}
	for name := range T.portals {
		T.deletePortal(name)
	}
	for pending, ok := T.pendingCloses.PopBack(); ok; pending, ok = T.pendingCloses.PopBack() {
		pending.Done()
	}
}

var _ middleware.Middleware = (*Server)(nil)
