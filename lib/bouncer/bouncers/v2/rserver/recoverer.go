package rserver

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func serverRead(ctx *bctx.Context) (zap.In, error) {
	for {
		in, err := ctx.ServerRead()
		if err != nil {
			return zap.In{}, err
		}
		switch in.Type() {
		case packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			continue
		default:
			return in, nil
		}
	}
}

func copyIn(ctx *bctx.Context) error {
	// send copy fail
	out := ctx.ServerWrite()
	out.Type(packets.CopyFail)
	out.String("client failed")
	if err := ctx.ServerSend(out); err != nil {
		return err
	}
	ctx.CopyIn = false
	return nil
}

func copyOut(ctx *bctx.Context) error {
	for {
		in, err := serverRead(ctx)
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.CopyData:
			continue
		case packets.CopyDone, packets.ErrorResponse:
			ctx.CopyOut = false
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func query(ctx *bctx.Context) error {
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
			continue
		case packets.CopyInResponse:
			ctx.CopyIn = true
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			ctx.CopyOut = true
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			ctx.Query = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func functionCall(ctx *bctx.Context) error {
	for {
		in, err := serverRead(ctx)
		if err != nil {
			return err
		}

		switch in.Type() {
		case packets.ErrorResponse, packets.FunctionCallResponse:
			continue
		case packets.ReadyForQuery:
			ctx.FunctionCall = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(in); !ok {
				return berr.ServerBadPacket
			}
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func sync(ctx *bctx.Context) error {
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
			continue
		case packets.CopyInResponse:
			ctx.CopyIn = true
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			ctx.CopyOut = true
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
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func eqp(ctx *bctx.Context) error {
	// send sync
	out := ctx.ServerWrite()
	out.Type(packets.Sync)
	if err := ctx.ServerSend(out); err != nil {
		return err
	}
	ctx.Sync = true

	// handle sync
	return sync(ctx)
}

func transaction(ctx *bctx.Context) error {
	// write Query('ABORT;')
	out := ctx.ServerWrite()
	out.Type(packets.Query)
	out.String("ABORT;")
	if err := ctx.ServerSend(out); err != nil {
		return err
	}
	ctx.Query = true

	// handle query
	return query(ctx)
}

func Recover(ctx *bctx.Context) error {
	if ctx.CopyOut {
		if err := copyOut(ctx); err != nil {
			return err
		}
	}
	if ctx.CopyIn {
		if err := copyIn(ctx); err != nil {
			return err
		}
	}
	if ctx.Query {
		if err := query(ctx); err != nil {
			return err
		}
	}
	if ctx.FunctionCall {
		if err := functionCall(ctx); err != nil {
			return err
		}
	}
	if ctx.Sync {
		if err := sync(ctx); err != nil {
			return err
		}
	}
	if ctx.EQP {
		if err := eqp(ctx); err != nil {
			return err
		}
	}
	if ctx.TxState != 'I' {
		if err := transaction(ctx); err != nil {
			return err
		}
	}
	return nil
}
