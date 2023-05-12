package eqp

import (
	"errors"

	"pggat2/lib/mw2"
	"pggat2/lib/util/ring"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Server struct {
	preparedStatements        map[string]PreparedStatement
	portals                   map[string]Portal
	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]

	peer *Client
}

func (T *Server) closePreparedStatement(ctx mw2.Context, target string) error {

}

func (T *Server) closePortal(ctx mw2.Context, target string) error {

}

func (T *Server) bindPreparedStatement(ctx mw2.Context, target string, preparedStatement PreparedStatement) error {
	if _, ok := T.preparedStatements[target]; ok {
		err := T.closePreparedStatement(ctx, target)
		if err != nil {
			return err
		}
	}
}

func (T *Server) bindPortal(ctx mw2.Context, target string, portal Portal) error {
	if _, ok := T.portals[target]; ok {
		err := T.closePortal(ctx, target)
		if err != nil {
			return err
		}
	}
}

func (T *Server) syncPreparedStatement(ctx mw2.Context, target string) error {

}

func (T *Server) syncPortal(ctx mw2.Context, target string) error {

}

func (T *Server) Send(ctx mw2.Context, out zap.Out) error {
	in := zap.OutToIn(out)
	switch in.Type() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		delete(T.preparedStatements, "")
		delete(T.portals, "")
	case packets.Parse, packets.Bind, packets.Close:
		// should've been caught by eqp.Client
		panic("unreachable")
	case packets.Describe:
		// ensure target exists
		which, target, ok := packets.ReadDescribe(in)
		if !ok {
			return errors.New("bad packet format")
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
			return errors.New("unknown describe target")
		}
	case packets.Execute:
		target, _, ok := packets.ReadExecute(in)
		if !ok {
			return errors.New("bad packet format")
		}
		// sync portal
		err := T.syncPortal(ctx, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (T *Server) Read(ctx mw2.Context, in zap.In) error {
	switch in.Type() {
	case packets.ParseComplete:
		ctx.Cancel()

		T.pendingPreparedStatements.PopFront()
	case packets.BindComplete:
		ctx.Cancel()

		T.pendingPortals.PopFront()
	case packets.CloseComplete:
		ctx.Cancel()

		// TODO(garet) Correctness: we could check this to make sure state is synced, but waiting for close is a pain
	case packets.ReadyForQuery:
		// all pending failed
		for pending, ok := T.pendingPreparedStatements.PopFront(); ok; pending, ok = T.pendingPreparedStatements.PopFront() {
			delete(T.preparedStatements, pending)
		}
		for pending, ok := T.pendingPortals.PopFront(); ok; pending, ok = T.pendingPortals.PopFront() {
			delete(T.portals, pending)
		}
	}
	return nil
}

var _ mw2.Middleware = (*Server)(nil)
