package bouncers

import (
	"log"

	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/berr"
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
			return ctx.ServerProxy(in)
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
			return ctx.ClientProxy(in)
		default:
			log.Println("a")
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
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			if err = ctx.ClientProxy(in); err != nil {
				return err
			}
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientProxy(in)
		default:
			log.Println("b")
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
		case packets.ReadyForQuery:
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientProxy(in)
		default:
			log.Println("c", in.Type())
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
		if ctx.TxState == 'I' {
			return nil
		}

		in, err := ctx.ClientRead()
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.Query:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			if err = query(ctx); err != nil {
				return err
			}
		case packets.FunctionCall:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			if err = functionCall(ctx); err != nil {
				return err
			}
		case packets.Sync:
			// TODO(garet) can this be turned into a phony call, not actually directed to server?
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			if err = sync(ctx); err != nil {
				return err
			}
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err = ctx.ServerProxy(in); err != nil {
				return err
			}
			if err = eqp(ctx); err != nil {
				return err
			}
		default:
			return berr.ClientUnexpectedPacket
		}
	}
}

func Bounce(client, server zap.ReadWriter) {
	ctx := bctx.MakeContext(client, server)
	ctx.TxState = 'T'
	err := transaction(&ctx)
	if err != nil {
		panic(err)
	}
}
