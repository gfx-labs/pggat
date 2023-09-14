package backends

import (
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/util/strutil"
)

func CopyIn(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	ctx.PeerWrite(packet)

	for {
		packet = ctx.PeerRead()
		if packet == nil {
			copyFail := packets.CopyFail{
				Reason: "peer failed",
			}
			return server.WritePacket(copyFail.IntoPacket())
		}

		switch packet.Type() {
		case packets.TypeCopyData:
			if err := server.WritePacket(packet); err != nil {
				return err
			}
		case packets.TypeCopyDone, packets.TypeCopyFail:
			return server.WritePacket(packet)
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func CopyOut(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	ctx.PeerWrite(packet)

	for {
		var err error
		packet, err = server.ReadPacket(true)
		if err != nil {
			return err
		}

		switch packet.Type() {
		case packets.TypeCopyData,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite(packet)
		case packets.TypeCopyDone, packets.TypeErrorResponse:
			ctx.PeerWrite(packet)
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func Query(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	if err := server.WritePacket(packet); err != nil {
		return err
	}

	for {
		var err error
		packet, err = server.ReadPacket(true)
		if err != nil {
			return err
		}

		switch packet.Type() {
		case packets.TypeCommandComplete,
			packets.TypeRowDescription,
			packets.TypeDataRow,
			packets.TypeEmptyQueryResponse,
			packets.TypeErrorResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite(packet)
		case packets.TypeCopyInResponse:
			if err = CopyIn(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeCopyOutResponse:
			if err = CopyOut(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(packet) {
				return ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite(packet)
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func QueryString(ctx *Context, server fed.ReadWriter, query string) error {
	q := packets.Query(query)
	return Query(ctx, server, q.IntoPacket())
}

func SetParameter(ctx *Context, server fed.ReadWriter, name strutil.CIString, value string) error {
	return QueryString(ctx, server, `SET `+strutil.Escape(name.String(), `"`)+` = `+strutil.Escape(value, `'`))
}

func FunctionCall(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	if err := server.WritePacket(packet); err != nil {
		return err
	}

	for {
		var err error
		packet, err = server.ReadPacket(true)
		if err != nil {
			return err
		}

		switch packet.Type() {
		case packets.TypeErrorResponse,
			packets.TypeFunctionCallResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite(packet)
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(packet) {
				return ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite(packet)
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func Sync(ctx *Context, server fed.ReadWriter) error {
	if err := server.WritePacket(fed.NewPacket(packets.TypeSync)); err != nil {
		return err
	}

	for {
		packet, err := server.ReadPacket(true)
		if err != nil {
			return err
		}

		switch packet.Type() {
		case packets.TypeParseComplete,
			packets.TypeBindComplete,
			packets.TypeCloseComplete,
			packets.TypeErrorResponse,
			packets.TypeRowDescription,
			packets.TypeNoData,
			packets.TypeParameterDescription,

			packets.TypeCommandComplete,
			packets.TypeDataRow,
			packets.TypeEmptyQueryResponse,
			packets.TypePortalSuspended,

			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite(packet)
		case packets.TypeCopyInResponse:
			if err = CopyIn(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeCopyOutResponse:
			if err = CopyOut(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(packet) {
				return ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite(packet)
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func EQP(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	if err := server.WritePacket(packet); err != nil {
		return err
	}

	for {
		packet = ctx.PeerRead()
		if packet == nil {
			return Sync(ctx, server)
		}

		switch packet.Type() {
		case packets.TypeSync:
			return Sync(ctx, server)
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := server.WritePacket(packet); err != nil {
				return err
			}
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func Transaction(ctx *Context, server fed.ReadWriter, packet fed.Packet) error {
	if ctx.TxState == '\x00' {
		ctx.TxState = 'I'
	}
	for {
		switch packet.Type() {
		case packets.TypeQuery:
			if err := Query(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeFunctionCall:
			if err := FunctionCall(ctx, server, packet); err != nil {
				return err
			}
		case packets.TypeSync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			rfq := packets.ReadyForQuery(ctx.TxState)
			ctx.PeerWrite(rfq.IntoPacket())
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := EQP(ctx, server, packet); err != nil {
				return err
			}
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}

		if ctx.TxState == 'I' {
			return nil
		}

		packet = ctx.PeerRead()
		if packet == nil {
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
