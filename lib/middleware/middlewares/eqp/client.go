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

func MakeClient() Client {
	return Client{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
	}
}

func (T *Client) Send(_ middleware.Context, out zap.Out) error {
	in := zap.OutToIn(out)
	switch in.Type() {
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
	case packets.ParseComplete, packets.BindComplete, packets.CloseComplete:
		// should've been caught by eqp.Server
		panic("unreachable")
	}
	return nil
}

func (T *Client) Read(ctx middleware.Context, in zap.In) error {
	switch in.Type() {
	case packets.Query:
		// clobber unnamed portal and unnamed prepared statement
		delete(T.preparedStatements, "")
		delete(T.portals, "")
	case packets.Parse:
		ctx.Cancel()

		destination, query, parameterDataTypes, ok := packets.ReadParse(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if destination != "" {
			if _, ok = T.preparedStatements[destination]; ok {
				return errors.New("prepared statement already exists")
			}
		}
		T.preparedStatements[destination] = PreparedStatement{
			Query:              query,
			ParameterDataTypes: parameterDataTypes,
		}

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

		destination, source, parameterFormatCodes, parameterValues, resultFormatCodes, ok := packets.ReadBind(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if destination != "" {
			if _, ok = T.portals[destination]; ok {
				return errors.New("portal already exists")
			}
		}
		if _, ok = T.preparedStatements[source]; !ok {
			return errors.New("prepared statement does not exist")
		}
		T.portals[destination] = Portal{
			Source:               source,
			ParameterFormatCodes: parameterFormatCodes,
			ParameterValues:      parameterValues,
			ResultFormatCodes:    resultFormatCodes,
		}

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
			delete(T.preparedStatements, target)
		case 'P':
			delete(T.portals, target)
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
		which, target, ok := packets.ReadDescribe(in)
		if !ok {
			return errors.New("bad packet format")
		}
		switch which {
		case 'S':
			if _, ok := T.preparedStatements[target]; !ok {
				return errors.New("prepared statement doesn't exist")
			}
		case 'P':
			if _, ok := T.portals[target]; !ok {
				return errors.New("portal doesn't exist")
			}
		default:
			return errors.New("unknown describe target")
		}
	case packets.Execute:
		target, _, ok := packets.ReadExecute(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if _, ok := T.portals[target]; !ok {
			return errors.New("portal doesn't exist")
		}
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
