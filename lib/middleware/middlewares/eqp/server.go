package eqp

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/ring"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type pendingClose interface {
	pendingClose()
}

type pendingClosePreparedStatement struct {
	target            string
	preparedStatement PreparedStatement
}

func (pendingClosePreparedStatement) pendingClose() {}

type pendingClosePortal struct {
	target string
	portal Portal
}

func (pendingClosePortal) pendingClose() {}

type Server struct {
	preparedStatements        map[string]PreparedStatement
	portals                   map[string]Portal
	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]
	pendingCloses             ring.Ring[pendingClose]

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

func (T *Server) closePreparedStatement(ctx middleware.Context, target string) error {
	out := T.buf.Write()
	packets.WriteClose(out, 'S', target)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	preparedStatement := T.preparedStatements[target]
	delete(T.preparedStatements, target)
	T.pendingCloses.PushBack(pendingClosePreparedStatement{
		target:            target,
		preparedStatement: preparedStatement,
	})
	return nil
}

func (T *Server) closePortal(ctx middleware.Context, target string) error {
	out := T.buf.Write()
	packets.WriteClose(out, 'P', target)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	portal := T.portals[target]
	delete(T.portals, target)
	T.pendingCloses.PushBack(pendingClosePortal{
		target: target,
		portal: portal,
	})
	return nil
}

func (T *Server) bindPreparedStatement(ctx middleware.Context, target string, preparedStatement PreparedStatement) error {
	if target != "" {
		if _, ok := T.preparedStatements[target]; ok {
			err := T.closePreparedStatement(ctx, target)
			if err != nil {
				return err
			}
		}
	}

	out := T.buf.Write()
	packets.WriteParse(out, target, preparedStatement.Query, preparedStatement.ParameterDataTypes)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	T.preparedStatements[target] = preparedStatement
	T.pendingPreparedStatements.PushBack(target)
	return nil
}

func (T *Server) bindPortal(ctx middleware.Context, target string, portal Portal) error {
	if target != "" {
		if _, ok := T.portals[target]; ok {
			err := T.closePortal(ctx, target)
			if err != nil {
				return err
			}
		}
	}

	out := T.buf.Write()
	packets.WriteBind(out, target, portal.Source, portal.ParameterFormatCodes, portal.ParameterValues, portal.ResultFormatCodes)
	err := ctx.Send(out)
	if err != nil {
		return err
	}

	T.portals[target] = portal
	T.pendingPortals.PushBack(target)
	return nil
}

func (T *Server) syncPreparedStatement(ctx middleware.Context, target string) error {
	// we can assume client has the prepared statement because it should be checked by eqp.Client
	expected := T.peer.preparedStatements[target]
	actual, ok := T.preparedStatements[target]
	if !ok || !expected.Equals(actual) {
		// clear all portals that use this prepared statement
		for name, portal := range T.portals {
			if portal.Source == target {
				err := T.closePortal(ctx, name)
				if err != nil {
					return err
				}
			}
		}
		return T.bindPreparedStatement(ctx, target, expected)
	}
	return nil
}

func (T *Server) syncPortal(ctx middleware.Context, target string) error {
	expected := T.peer.portals[target]
	err := T.syncPreparedStatement(ctx, expected.Source)
	if err != nil {
		return err
	}
	actual, ok := T.portals[target]
	if !ok || !expected.Equals(actual) {
		return T.bindPortal(ctx, target, expected)
	}
	return nil
}

func (T *Server) Send(ctx middleware.Context, out zap.Out) error {
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

		T.pendingCloses.PopFront()
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if state == 'I' {
			// clobber all portals
			for name := range T.portals {
				delete(T.portals, name)
			}
		}
		// all pending failed
		for pending, ok := T.pendingPreparedStatements.PopBack(); ok; pending, ok = T.pendingPreparedStatements.PopBack() {
			delete(T.preparedStatements, pending)
		}
		for pending, ok := T.pendingPortals.PopBack(); ok; pending, ok = T.pendingPortals.PopBack() {
			delete(T.portals, pending)
		}
		for pending, ok := T.pendingCloses.PopBack(); ok; pending, ok = T.pendingCloses.PopBack() {
			switch p := pending.(type) {
			case pendingClosePortal:
				T.portals[p.target] = p.portal
			case pendingClosePreparedStatement:
				T.preparedStatements[p.target] = p.preparedStatement
			default:
				panic("what")
			}
		}
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
