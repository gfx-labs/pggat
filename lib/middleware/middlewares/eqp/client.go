package eqp

import (
	"errors"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/middleware"
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
	delete(T.preparedStatements, name)
}

func (T *Client) deletePortal(name string) {
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

func (T *Client) Write(_ middleware.Context, packet fed.Packet) error {
	switch packet.Type() {
	case packets.TypeReadyForQuery:
		var readyForQuery packets.ReadyForQuery
		if !readyForQuery.ReadFromPacket(packet) {
			return errors.New("bad packet format a")
		}
		if readyForQuery == 'I' {
			// clobber all named portals
			for name := range T.portals {
				T.deletePortal(name)
			}
		}
	case packets.TypeParseComplete, packets.TypeBindComplete, packets.TypeCloseComplete:
		// should've been caught by eqp.Server
		panic("unreachable")
	}
	return nil
}

func (T *Client) Read(ctx middleware.Context, packet fed.Packet) error {
	switch packet.Type() {
	case packets.TypeQuery:
		// clobber unnamed portal and unnamed prepared statement
		T.deletePreparedStatement("")
		T.deletePortal("")
	case packets.TypeParse:
		ctx.Cancel()

		destination, preparedStatement, ok := ReadParse(packet)
		if !ok {
			return errors.New("bad packet format b")
		}

		T.preparedStatements[destination] = preparedStatement

		// send parse complete
		packet = fed.NewPacket(packets.TypeParseComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.TypeBind:
		ctx.Cancel()

		destination, portal, ok := ReadBind(packet)
		if !ok {
			return errors.New("bad packet format c")
		}

		T.portals[destination] = portal

		// send bind complete
		packet = fed.NewPacket(packets.TypeParseComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.TypeClose:
		ctx.Cancel()

		var p packets.Close
		if !p.ReadFromPacket(packet) {
			return errors.New("bad packet format d")
		}
		switch p.Which {
		case 'S':
			T.deletePreparedStatement(p.Target)
		case 'P':
			T.deletePortal(p.Target)
		default:
			return errors.New("bad packet format e")
		}

		// send close complete
		packet = fed.NewPacket(packets.TypeCloseComplete)
		err := ctx.Write(packet)
		if err != nil {
			return err
		}
	case packets.TypeDescribe:
		// ensure target exists
		var describe packets.Describe
		if !describe.ReadFromPacket(packet) {
			return errors.New("bad packet format f")
		}
		switch describe.Which {
		case 'S', 'P':
			// ok
		default:
			return errors.New("unknown describe target")
		}
	case packets.TypeExecute:
		var execute packets.Execute
		if !execute.ReadFromPacket(packet) {
			return errors.New("bad packet format g")
		}
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
