package backends

import (
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func CopyIn(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	ctx.PeerWrite(packet)

	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet = zap.NewPacket()
		if !ctx.PeerRead(packet) {
			packets.WriteCopyFail(packet, "peer failed")
			pkts.Append(packet)
			return server.WriteV(pkts)
		}

		switch packet.ReadType() {
		case packets.CopyData:
			pkts.Append(packet)
		case packets.CopyDone, packets.CopyFail:
			pkts.Append(packet)
			return server.WriteV(pkts)
		default:
			packet.Done()
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func CopyOut(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	ctx.PeerWrite(packet)

	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet = zap.NewPacket()
		if err := server.Read(packet); err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.CopyData,
			packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			pkts.Append(packet)
		case packets.CopyDone, packets.ErrorResponse:
			pkts.Append(packet)
			ctx.PeerWriteV(pkts)
			return nil
		default:
			packet.Done()
			return ErrUnexpectedPacket
		}
	}
}

func Query(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	if err := server.Write(packet); err != nil {
		return err
	}

	pkts := zap.NewPackets()
	defer pkts.Done()

	for {
		packet = zap.NewPacket()

		err := server.Read(packet)
		if err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.CommandComplete,
			packets.RowDescription,
			packets.DataRow,
			packets.EmptyQueryResponse,
			packets.ErrorResponse,
			packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			pkts.Append(packet)
		case packets.CopyInResponse:
			ctx.PeerWriteV(pkts)
			pkts.Clear()
			if err = CopyIn(ctx, server, packet); err != nil {
				packet.Done()
				return err
			}
			packet.Done()
		case packets.CopyOutResponse:
			ctx.PeerWriteV(pkts)
			pkts.Clear()
			if err = CopyOut(ctx, server, packet); err != nil {
				packet.Done()
				return err
			}
			packet.Done()
		case packets.ReadyForQuery:
			var ok bool
			ctx.TxState, ok = packets.ReadReadyForQuery(packet.Read())
			if !ok {
				packet.Done()
				return ErrBadFormat
			}
			pkts.Append(packet)
			ctx.PeerWriteV(pkts)
			return nil
		default:
			packet.Done()
			return ErrUnexpectedPacket
		}
	}
}

func QueryString(ctx *Context, server zap.ReadWriter, query string) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteQuery(packet, query)
	return Query(ctx, server, packet)
}

func FunctionCall(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	if err := server.Write(packet); err != nil {
		return err
	}

	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet = zap.NewPacket()
		if err := server.Read(packet); err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.ErrorResponse,
			packets.FunctionCallResponse,
			packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			pkts.Append(packet)
		case packets.ReadyForQuery:
			var ok bool
			ctx.TxState, ok = packets.ReadReadyForQuery(packet.Read())
			if !ok {
				packet.Done()
				return ErrBadFormat
			}
			pkts.Append(packet)
			ctx.PeerWriteV(pkts)
			return nil
		default:
			packet.Done()
			return ErrUnexpectedPacket
		}
	}
}

func Sync(ctx *Context, server zap.ReadWriter) error {
	var err error
	func() {
		packet := zap.NewPacket()
		defer packet.Done()
		packet.WriteType(packets.Sync)
		err = server.Write(packet)
	}()
	if err != nil {
		return err
	}

	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet := zap.NewPacket()
		if err = server.Read(packet); err != nil {
			packet.Done()
			return err
		}

		switch packet.ReadType() {
		case packets.ParseComplete,
			packets.BindComplete,
			packets.ErrorResponse,
			packets.RowDescription,
			packets.NoData,
			packets.ParameterDescription,

			packets.CommandComplete,
			packets.DataRow,
			packets.EmptyQueryResponse,
			packets.PortalSuspended,

			packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			pkts.Append(packet)
		case packets.CopyInResponse:
			ctx.PeerWriteV(pkts)
			pkts.Clear()
			if err = CopyIn(ctx, server, packet); err != nil {
				packet.Done()
				return err
			}
			packet.Done()
		case packets.CopyOutResponse:
			ctx.PeerWriteV(pkts)
			pkts.Clear()
			if err = CopyOut(ctx, server, packet); err != nil {
				packet.Done()
				return err
			}
			packet.Done()
		case packets.ReadyForQuery:
			var ok bool
			ctx.TxState, ok = packets.ReadReadyForQuery(packet.Read())
			if !ok {
				packet.Done()
				return ErrBadFormat
			}
			pkts.Append(packet)
			ctx.PeerWriteV(pkts)
			return nil
		default:
			packet.Done()
			return ErrUnexpectedPacket
		}
	}
}

func EQP(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	if err := server.Write(packet); err != nil {
		return err
	}

	pkts := zap.NewPackets()
	defer pkts.Done()
	for {
		packet = zap.NewPacket()
		if !ctx.PeerRead(packet) {
			if err := server.WriteV(pkts); err != nil {
				return err
			}
			return Sync(ctx, server)
		}

		switch packet.ReadType() {
		case packets.Sync:
			packet.Done()
			if err := server.WriteV(pkts); err != nil {
				return err
			}
			return Sync(ctx, server)
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			pkts.Append(packet)
		default:
			packet.Done()
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func Transaction(ctx *Context, server zap.ReadWriter, packet *zap.Packet) error {
	for {
		switch packet.ReadType() {
		case packets.Query:
			if err := Query(ctx, server, packet); err != nil {
				return err
			}
		case packets.FunctionCall:
			if err := FunctionCall(ctx, server, packet); err != nil {
				return err
			}
		case packets.Sync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			packets.WriteReadyForQuery(packet, ctx.TxState)
			ctx.PeerWrite(packet)
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err := EQP(ctx, server, packet); err != nil {
				return err
			}
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
			continue
		}

		if ctx.TxState == 'I' || ctx.TxState == '\x00' {
			return nil
		}

		if !ctx.PeerRead(packet) {
			// abort tx
			err := QueryString(ctx, server, "ABORT;")
			if err != nil {
				return err
			}

			if ctx.TxState != 'I' {
				return ErrUnexpectedPacket
			}
			return nil
		}
	}
}
