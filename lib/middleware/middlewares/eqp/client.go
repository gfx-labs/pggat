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

	middleware.Nil
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

func (T *Client) Write(_ middleware.Context, packet *zap.Packet) error {
	read := packet.Read()
	switch read.ReadType() {
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(&read)
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

func (T *Client) Read(ctx middleware.Context, packet *zap.Packet) error {
	read := packet.Read()
	switch read.ReadType() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		T.deletePreparedStatement("")
		T.deletePortal("")
	case packets.Parse:
		ctx.Cancel()

		destination, preparedStatement, ok := ReadParse(&read)
		if !ok {
			return errors.New("bad packet format")
		}

		T.preparedStatements[destination] = preparedStatement

		// send parse complete
		packet.WriteType(packets.ParseComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.Bind:
		ctx.Cancel()

		destination, portal, ok := ReadBind(&read)
		if !ok {
			return errors.New("bad packet format")
		}

		T.portals[destination] = portal

		// send bind complete
		packet.WriteType(packets.BindComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.Close:
		ctx.Cancel()

		which, target, ok := packets.ReadClose(&read)
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
		packet.WriteType(packets.CloseComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.Describe:
		// ensure target exists
		which, _, ok := packets.ReadDescribe(&read)
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
		_, _, ok := packets.ReadExecute(&read)
		if !ok {
			return errors.New("bad packet format")
		}
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
