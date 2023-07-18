package eqp

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal
}

func NewClient() *Client {
	return &Client{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
	}
}

func (T *Client) deletePreparedStatement(name string) {
	preparedStatement, ok := T.preparedStatements[name]
	if !ok {
		return
	}
	preparedStatement.Done()
	delete(T.preparedStatements, name)
}

func (T *Client) deletePortal(name string) {
	portal, ok := T.portals[name]
	if !ok {
		return
	}
	portal.Done()
	delete(T.portals, name)
}

func (T *Client) Done() {
	for name := range T.preparedStatements {
		T.deletePreparedStatement(name)
	}
	for name := range T.portals {
		T.deletePortal(name)
	}
}

func (T *Client) Write(_ middleware.Context, in zap.Inspector) error {
	switch in.Type() {
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if state == 'I' {
			// clobber all named portals
			for name := range T.portals {
				T.deletePortal(name)
			}
		}
	case packets.ParseComplete, packets.BindComplete, packets.CloseComplete:
		// should've been caught by eqp.Server
		panic("unreachable")
	}
	return nil
}

func (T *Client) Read(ctx middleware.Context, in zap.Inspector) error {
	switch in.Type() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		T.deletePreparedStatement("")
		T.deletePortal("")
	case packets.Parse:
		ctx.Cancel()

		destination, preparedStatement, ok := ReadParse(in)
		if !ok {
			return errors.New("bad packet format")
		}

		T.preparedStatements[destination] = preparedStatement

		// send parse complete
		out := zap.InToOut(in)
		out.Reset()
		out.Type(packets.ParseComplete)
		err := ctx.Send(out)
		if err != nil {
			return err
		}
	case packets.Bind:
		ctx.Cancel()

		destination, portal, ok := ReadBind(in)
		if !ok {
			return errors.New("bad packet format")
		}

		T.portals[destination] = portal

		// send bind complete
		out := zap.InToOut(in)
		out.Reset()
		out.Type(packets.BindComplete)
		err := ctx.Send(out)
		if err != nil {
			return err
		}
	case packets.Close:
		ctx.Cancel()

		which, target, ok := packets.ReadClose(in)
		if !ok {
			return errors.New("bad packet format")
		}
		switch which {
		case 'S':
			T.deletePreparedStatement(target)
		case 'P':
			T.deletePortal(target)
		default:
			return errors.New("bad packet format")
		}

		// send close complete
		out := zap.InToOut(in)
		out.Reset()
		out.Type(packets.CloseComplete)
		err := ctx.Send(out)
		if err != nil {
			return err
		}
	case packets.Describe:
		// ensure target exists
		which, _, ok := packets.ReadDescribe(in)
		if !ok {
			return errors.New("bad packet format")
		}
		switch which {
		case 'S', 'P':
			// ok
		default:
			return errors.New("unknown describe target")
		}
	case packets.Execute:
		_, _, ok := packets.ReadExecute(in)
		if !ok {
			return errors.New("bad packet format")
		}
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
