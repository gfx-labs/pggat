package bouncers

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/bouncer/bouncers/v2/rserver"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func serverRead(ctx *bctx.Context, packet *zap.Packet) berr.Error {
	for {
		err := ctx.ServerRead(packet)
		if err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			if err = ctx.ClientWrite(packet); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func copyIn(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := ctx.ClientRead(packet)
		if err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.CopyData:
			pkts.Append(packet)
		case packets.CopyDone, packets.CopyFail:
			pkts.Append(packet)
			ctx.CopyIn = false
			return ctx.ServerWriteV(pkts)
		default:
			packet.Done()
			return berr.ClientUnexpectedPacket
		}
	}
}

func copyOut(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := serverRead(ctx, packet)
		if err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.CopyData:
			pkts.Append(packet)
		case packets.CopyDone, packets.ErrorResponse:
			pkts.Append(packet)
			ctx.CopyOut = false
			return ctx.ClientWriteV(pkts)
		default:
			packet.Done()
			return berr.ServerUnexpectedPacket
		}
	}
}

func query(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := serverRead(ctx, packet)
		if err != nil {
			packet.Done()
			return err
		}

		read := packet.Read()

		switch read.ReadType() {
		case packets.CommandComplete,
			packets.RowDescription,
			packets.DataRow,
			packets.EmptyQueryResponse,
			packets.ErrorResponse:
			pkts.Append(packet)
		case packets.CopyInResponse:
			pkts.Append(packet)
			ctx.CopyIn = true
			if err = ctx.ClientWriteV(pkts); err != nil {
				return err
			}
			pkts.Clear()
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			pkts.Append(packet)
			ctx.CopyOut = true
			if err = ctx.ClientWriteV(pkts); err != nil {
				return err
			}
			pkts.Clear()
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			pkts.Append(packet)
			ctx.Query = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientWriteV(pkts)
		default:
			packet.Done()
			return berr.ServerUnexpectedPacket
		}
	}
}

func functionCall(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := serverRead(ctx, packet)
		if err != nil {
			packet.Done()
			return err
		}

		read := packet.Read()

		switch read.ReadType() {
		case packets.ErrorResponse, packets.FunctionCallResponse:
			pkts.Append(packet)
		case packets.ReadyForQuery:
			pkts.Append(packet)
			ctx.FunctionCall = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientWriteV(pkts)
		default:
			packet.Done()
			return berr.ServerUnexpectedPacket
		}
	}
}

func sync(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := serverRead(ctx, packet)
		if err != nil {
			packet.Done()
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
			pkts.Append(packet)
		case packets.CopyInResponse:
			ctx.CopyIn = true
			pkts.Append(packet)
			if err = ctx.ClientWriteV(pkts); err != nil {
				return err
			}
			pkts.Clear()
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.CopyOutResponse:
			ctx.CopyOut = true
			pkts.Append(packet)
			if err = ctx.ClientWriteV(pkts); err != nil {
				return err
			}
			pkts.Clear()
			if err = copyOut(ctx); err != nil {
				return err
			}
		case packets.ReadyForQuery:
			pkts.Append(packet)
			ctx.Sync = false
			ctx.EQP = false
			var ok bool
			if ctx.TxState, ok = packets.ReadReadyForQuery(&read); !ok {
				return berr.ServerBadPacket
			}
			return ctx.ClientWriteV(pkts)
		default:
			packet.Done()
			return berr.ServerUnexpectedPacket
		}
	}
}

func eqp(ctx *bctx.Context) berr.Error {
	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		err := ctx.ClientRead(packet)
		if err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.Sync:
			pkts.Append(packet)
			ctx.Sync = true
			if err = ctx.ServerWriteV(pkts); err != nil {
				return err
			}
			pkts.Clear()
			ctx.Sync = true
			return sync(ctx)
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			pkts.Append(packet)
		default:
			packet.Done()
			return berr.ClientUnexpectedPacket
		}
	}
}

func transaction(ctx *bctx.Context) berr.Error {
	packet := zap.NewPacket()
	defer packet.Done()
	for {
		err := ctx.ClientRead(packet)
		if err != nil {
			return err
		}

		switch packet.ReadType() {
		case packets.Query:
			if err = ctx.ServerWrite(packet); err != nil {
				return err
			}
			ctx.Query = true
			if err = query(ctx); err != nil {
				return err
			}
		case packets.FunctionCall:
			if err = ctx.ServerWrite(packet); err != nil {
				return err
			}
			ctx.FunctionCall = true
			if err = functionCall(ctx); err != nil {
				return err
			}
		case packets.Sync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			packets.WriteReadyForQuery(packet, ctx.TxState)
			if err = ctx.ClientWrite(packet); err != nil {
				return err
			}
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err = ctx.ServerWrite(packet); err != nil {
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

func clientFail(ctx *bctx.Context, err perror.Error) {
	// send fatal error to client
	packet := zap.NewPacket()
	packets.WriteErrorResponse(packet, err)
	_ = ctx.ClientWrite(packet)
}

func Bounce(client, server zap.ReadWriter) (clientError error, serverError error) {
	ctx := bctx.MakeContext(client, server)
	err := transaction(&ctx)
	if err != nil {
		switch e := err.(type) {
		case berr.Client:
			clientError = e
			serverError = rserver.Recover(&ctx)
			clientFail(&ctx, perror.Wrap(clientError))
		case berr.Server:
			serverError = e
			clientFail(&ctx, perror.Wrap(serverError))
		default:
			panic("unreachable")
		}
	}

	return
}
