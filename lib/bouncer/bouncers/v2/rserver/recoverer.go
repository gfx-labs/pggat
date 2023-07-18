package rserver

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func serverRead(ctx *bctx.Context, packet *zap.Packet) error {
	for {
		err := ctx.ServerRead(packet)
		if err != nil {
			return err
		}

		switch packet.ReadType() {
		case packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			continue
		default:
			return nil
		}
	}
}

func copyIn(ctx *bctx.Context) error {
	// send copy fail
	packet := zap.NewPacket()
	defer packet.Done()
	packet.WriteType(packets.CopyFail)
	packet.WriteString("client failed")
	if err := ctx.ServerWrite(packet); err != nil {
		return err
	}
	ctx.CopyIn = false
	return nil
}

func copyOut(ctx *bctx.Context) error {
	packet := zap.NewPacket()
	defer packet.Done()
	for {
		err := serverRead(ctx, packet)
		if err != nil {
			return err
		}

		switch packet.ReadType() {
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
	packet := zap.NewPacket()
	defer packet.Done()
	for {
		err := serverRead(ctx, packet)
		if err != nil {
			return err
		}

		read := packet.Read()

		switch read.ReadType() {
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
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
				return berr.ServerBadPacket
			}
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func functionCall(ctx *bctx.Context) error {
	packet := zap.NewPacket()
	defer packet.Done()
	for {
		err := serverRead(ctx, packet)
		if err != nil {
			return err
		}

		read := packet.Read()

		switch read.ReadType() {
		case packets.ErrorResponse, packets.FunctionCallResponse:
			continue
		case packets.ReadyForQuery:
			ctx.FunctionCall = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
				return berr.ServerBadPacket
			}
			return nil
		default:
			return berr.ServerUnexpectedPacket
		}
	}
}

func sync(ctx *bctx.Context) error {
	packet := zap.NewPacket()
	defer packet.Done()
	for {
		err := serverRead(ctx, packet)
		if err != nil {
			return err
		}

		read := packet.Read()

		switch read.ReadType() {
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
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
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
	packet := zap.NewPacket()
	defer packet.Done()
	packet.WriteType(packets.Sync)
	if err := ctx.ServerWrite(packet); err != nil {
		return err
	}
	ctx.Sync = true

	// handle sync
	return sync(ctx)
}

func transaction(ctx *bctx.Context) error {
	// write Query('ABORT;')
	packet := zap.NewPacket()
	defer packet.Done()
	packet.WriteType(packets.Query)
	packet.WriteString("ABORT;")
	if err := ctx.ServerWrite(packet); err != nil {
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
