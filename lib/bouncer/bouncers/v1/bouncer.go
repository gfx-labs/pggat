package bouncers

import (
	"errors"
	"log"

	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type queryContext struct {
	*transactionContext
	done bool
}

func (T *queryContext) queryDone() {
	T.done = true
}

func query0(ctx *queryContext) Error {
	in, err := ctx.readServer()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CommandComplete,
		packets.RowDescription,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.ErrorResponse,
		packets.NoticeResponse,
		packets.ParameterStatus:
		return ctx.sendClient(zap.InToOut(in))
	case packets.CopyInResponse:
		// return copyIn(ctx, in)
		return nil
	case packets.CopyOutResponse:
		// return copyOut(ctx, in)
		return nil
	case packets.ReadyForQuery:
		state, ok := packets.ReadReadyForQuery(in)
		if !ok {
			return makeClientError(packets.ErrBadFormat)
		}
		err = ctx.sendClient(zap.InToOut(in))
		if err != nil {
			return err
		}
		ctx.queryDone()
		if state == 'I' {
			ctx.transactionDone()
		}
		return nil
	default:
		return makeServerError(errors.New("protocol error"))
	}
}

func query(c *transactionContext, in zap.In) Error {
	// send in (initial query) to server
	err := c.sendServer(zap.InToOut(in))
	if err != nil {
		return err
	}

	ctx := queryContext{
		transactionContext: c,
	}
	for !ctx.done {
		err = query0(&ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

type transactionContext struct {
	*context
	done bool
}

func (T *transactionContext) transactionDone() {
	T.done = true
}

func transaction0(ctx *transactionContext) Error {
	in, err := ctx.readClient()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.Query:
		return query(ctx, in)
	case packets.FunctionCall:
		// return functionCall(ctx, in)
		return nil
	case packets.Sync, packets.Parse, packets.Bind, packets.Describe, packets.Execute:
		// return eqp(ctx, in)
		return nil
	default:
		return makeClientError(perror.New(
			perror.ERROR,
			perror.FeatureNotSupported,
			"unsupported operation",
		))
	}
}

func transaction(c *context) Error {
	ctx := transactionContext{
		context: c,
	}
	for !ctx.done {
		err := transaction0(&ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

type context struct {
	client, server zap.ReadWriter
}

func (T *context) readClient() (zap.In, Error) {
	in, err := T.client.Read()
	if err != nil {
		return zap.In{}, wrapClientError(err)
	}
	return in, nil
}

func (T *context) readServer() (zap.In, Error) {
	in, err := T.server.Read()
	if err != nil {
		return zap.In{}, makeServerError(err)
	}
	return in, nil
}

func (T *context) sendClient(out zap.Out) Error {
	err := T.client.Send(out)
	if err != nil {
		return wrapClientError(err)
	}
	return nil
}

func (T *context) sendServer(out zap.Out) Error {
	err := T.server.Send(out)
	if err != nil {
		return makeServerError(err)
	}
	return nil
}

func Bounce(client, server zap.ReadWriter) {
	ctx := context{
		client: client,
		server: server,
	}
	err := transaction(&ctx)
	if err != nil {
		// TODO(garet) handle error
		log.Println(err)
	}
}
