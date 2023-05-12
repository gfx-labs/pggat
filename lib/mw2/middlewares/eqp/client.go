package eqp

import (
	"errors"

	"pggat2/lib/mw2"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal
}

func (T *Client) Send(_ mw2.Context, out zap.Out) error {
	in := zap.OutToIn(out)
	switch in.Type() {
	case packets.ParseComplete, packets.BindComplete, packets.CloseComplete:
		// should've been caught by eqp.Server
		panic("unreachable")
	}
	return nil
}

func (T *Client) Read(ctx mw2.Context, in zap.In) error {
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

		// TODO(garet) we should read Describe and Execute to check if target exists
	}
	return nil
}

var _ mw2.Middleware = (*Client)(nil)
