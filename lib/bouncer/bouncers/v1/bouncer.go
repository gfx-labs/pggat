package bouncers

import (
	"log"

	"pggat2/lib/bouncer/bouncers/v1/bctx"
	"pggat2/lib/bouncer/bouncers/v1/berr"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

// serverRead is a wrapper for bctx.Context's ServerRead but it handles async operations
func serverRead(ctx *bctx.Context) (zap.In, berr.Error) {
	for {
		in, err := ctx.ServerRead()
		if err != nil {
			return zap.In{}, err
		}
		switch in.Type() {
		case packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			err = ctx.ClientProxy(in)
			if err != nil {
				return zap.In{}, err
			}
		default:
			return in, nil
		}
	}
}

func readyForQuery(ctx *bctx.Context, in zap.In) berr.Error {
	state, ok := packets.ReadReadyForQuery(in)
	if !ok {
		return berr.ServerBadPacket
	}
	ctx.EndEQP()
	if state == 'I' {
		ctx.EndTransaction()
	}
	return nil
}

func copyIn0(ctx *bctx.Context) berr.Error {
	in, err := ctx.ClientRead()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CopyData:
		return ctx.ServerProxy(in)
	case packets.CopyDone, packets.CopyFail:
		ctx.EndCopyIn()
		return ctx.ServerProxy(in)
	default:
		return berr.ClientProtocolError
	}
}

func copyIn(ctx *bctx.Context) berr.Error {
	ctx.BeginCopyIn()

	for ctx.InCopyIn() {
		err := copyIn0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyOut0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CopyData:
		return ctx.ClientProxy(in)
	case packets.CopyDone, packets.ErrorResponse:
		ctx.EndCopyOut()
		return ctx.ClientProxy(in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func copyOut(ctx *bctx.Context) berr.Error {
	ctx.BeginCopyOut()

	for ctx.InCopyOut() {
		err := copyOut0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func query0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CommandComplete,
		packets.RowDescription,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.ErrorResponse:
		return ctx.ClientProxy(in)
	case packets.CopyInResponse:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		return copyIn(ctx)
	case packets.CopyOutResponse:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		return copyOut(ctx)
	case packets.ReadyForQuery:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		ctx.EndQuery()
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func query(ctx *bctx.Context) berr.Error {
	ctx.BeginQuery()

	for ctx.InQuery() {
		err := query0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func functionCall0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.ErrorResponse, packets.FunctionCallResponse:
		return ctx.ClientProxy(in)
	case packets.ReadyForQuery:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		ctx.EndFunctionCall()
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func functionCall(ctx *bctx.Context) berr.Error {
	ctx.BeginFunctionCall()

	for ctx.InFunctionCall() {
		err := functionCall0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func sync0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.ParseComplete,
		packets.BindComplete,
		packets.ErrorResponse,
		packets.RowDescription,
		packets.NoData,
		packets.ParameterDescription,

		packets.CommandComplete,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.PortalSuspended:
		return ctx.ClientProxy(in)
	case packets.ReadyForQuery:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		ctx.EndSync()
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func sync(ctx *bctx.Context) berr.Error {
	ctx.BeginSync()

	for ctx.InSync() {
		err := sync0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func transaction0(ctx *bctx.Context) berr.Error {
	in, err := ctx.ClientRead()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.Query:
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		return query(ctx)
	case packets.FunctionCall:
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		return functionCall(ctx)
	case packets.Sync:
		if !ctx.InEQP() {
			ctx.BeginEQP()
		}
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		return sync(ctx)
	case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
		if !ctx.InEQP() {
			ctx.BeginEQP()
		}
		return ctx.ServerProxy(in)
	default:
		return berr.ClientProtocolError
	}
}

func transaction(ctx *bctx.Context) berr.Error {
	ctx.BeginTransaction()

	for ctx.InTransaction() {
		err := transaction0(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func Bounce(client, server zap.ReadWriter) {
	ctx := bctx.MakeContext(client, server, 0) // TODO(garet) make this configurable
	defer ctx.Done()
	err := transaction(&ctx)
	if err != nil {
		switch e := err.(type) {
		case berr.Client:
			// send to client
			out := client.Write()
			packets.WriteErrorResponse(out, e.Error)
			_ = client.Send(out)
		case berr.Server:
			log.Println("server error", e.Error)
		default:
			// unhandled error
			panic(err)
		}
	}
}
