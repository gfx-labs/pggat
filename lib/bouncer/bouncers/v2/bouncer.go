package bouncers

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/bouncer/bouncers/v2/rserver"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

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
			if err = ctx.ClientProxy(in); err != nil {
				return zap.In{}, err
			}
		default:
			return in, nil
		}
	}
}

func copyIn(ctx *bctx.Context) berr.Error {
	for {
		in, err := ctx.ClientRead()
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.CopyData:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
		case packets.CopyDone, packets.CopyFail:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			ctx.CopyIn = false
			return nil
		default:
			return berr.ClientUnexpectedPacket
		}
	}
}

func copyOut(ctx *bctx.Context) berr.Error {
	for {
		in, err := serverRead(ctx)
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.CopyData:
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
		case packets.CopyDone, packets.ErrorResponse:
			ctx.CopyOut = false
			return ctx.ClientProxy(in)
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func query(ctx *bctx.Context) berr.Error {
	for {
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
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
		case packets.CopyInResponse:
			ctx.CopyIn = true
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			ctx.CopyOut = true
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			ctx.Query = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientProxy(in)
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func functionCall(ctx *bctx.Context) berr.Error {
	for {
		in, err := serverRead(ctx)
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.ErrorResponse, packets.FunctionCallResponse:
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			ctx.FunctionCall = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientProxy(in)
		}
	}
}

func sync(ctx *bctx.Context) berr.Error {
	for {
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
			err = ctx.ClientProxy(in)
			if err != nil {
				return err
			}
		case packets.CopyInResponse:
			ctx.CopyIn = true
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			ctx.CopyOut = true
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			ctx.Sync = false
			ctx.EQP = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientProxy(in)
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func eqp(ctx *bctx.Context) berr.Error {
	for {
		in, err := ctx.ClientRead()
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.Sync:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			ctx.Sync = true
			return sync(ctx)
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
		default:
			return berr.ClientUnexpectedPacket
		}
	}
}

func transaction(ctx *bctx.Context) berr.Error {
	for {
		in, err := ctx.ClientRead()
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.Query:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			ctx.Query = true
			if err = query(ctx); err != nil {
				return err
			}
		case packets.FunctionCall:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			ctx.FunctionCall = true
			if err = functionCall(ctx); err != nil {
				return err
			}
		case packets.Sync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			out := zap.InToOut(in)
			packets.WriteReadyForQuery(out, ctx.TxState)
			if err = ctx.ClientSend(out); err != nil {
				return err
			}
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			ctx.EQP = true
			if err = eqp(ctx); err != nil {
				return err
			}
		default:
			return berr.ClientUnexpectedPacket
		}

		if ctx.TxState == 'I' {
			return nil
		}
	}
}

func clientError(ctx *bctx.Context, err error) {
	// send fatal error to client
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, perror.New(
		perror.FATAL,
		perror.ProtocolViolation,
		err.Error(),
	))
	_ = ctx.ClientSend(out)
}

func serverError(ctx *bctx.Context, err error) {
	panic("server error: " + err.Error())
}

func Bounce(client, server zap.ReadWriter) {
	ctx := bctx.MakeContext(client, server)
	err := transaction(&ctx)
	if err != nil {
		switch e := err.(type) {
		case berr.Client:
			clientError(&ctx, e)
			if err2 := rserver.Recover(&ctx); err2 != nil {
				serverError(&ctx, err2)
			}
		case berr.Server:
			serverError(&ctx, e)
			clientError(&ctx, e)
		default:
			panic("unreachable")
		}
	}
}
