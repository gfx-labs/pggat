package backends

import (
	"fmt"

	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func CopyIn(ctx *Context) error {
	ctx.PeerWrite()

	for {
		if !ctx.PeerRead() {
			copyFail := packets.CopyFail{
				Reason: "peer failed",
			}
			ctx.Packet = copyFail.IntoPacket(ctx.Packet)
			return ctx.ServerWrite()
		}

		switch ctx.Packet.Type() {
		case packets.TypeCopyData:
			if err := ctx.ServerWrite(); err != nil {
				return err
			}
		case packets.TypeCopyDone, packets.TypeCopyFail:
			return ctx.ServerWrite()
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func CopyOut(ctx *Context) error {
	ctx.PeerWrite()

	for {
		err := ctx.ServerRead()
		if err != nil {
			return err
		}

		switch ctx.Packet.Type() {
		case packets.TypeCopyData,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite()
		case packets.TypeCopyDone, packets.TypeErrorResponse:
			ctx.PeerWrite()
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func Query(ctx *Context) error {
	if err := ctx.ServerWrite(); err != nil {
		return err
	}

	for {
		err := ctx.ServerRead()
		if err != nil {
			return err
		}

		switch ctx.Packet.Type() {
		case packets.TypeCommandComplete,
			packets.TypeRowDescription,
			packets.TypeDataRow,
			packets.TypeEmptyQueryResponse,
			packets.TypeErrorResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite()
		case packets.TypeCopyInResponse:
			if err = CopyIn(ctx); err != nil {
				return err
			}
		case packets.TypeCopyOutResponse:
			if err = CopyOut(ctx); err != nil {
				return err
			}
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(ctx.Packet) {
				return ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite()
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func QueryString(ctx *Context, query string) error {
	q := packets.Query(query)
	ctx.Packet = q.IntoPacket(ctx.Packet)
	return Query(ctx)
}

func SetParameter(ctx *Context, name strutil.CIString, value string) error {
	return QueryString(
		ctx,
		fmt.Sprintf(`SET "%s" = '%s'`, strutil.Escape(name.String(), '"'), strutil.Escape(value, '\'')),
	)
}

func FunctionCall(ctx *Context) error {
	if err := ctx.ServerWrite(); err != nil {
		return err
	}

	for {
		err := ctx.ServerRead()
		if err != nil {
			return err
		}

		switch ctx.Packet.Type() {
		case packets.TypeErrorResponse,
			packets.TypeFunctionCallResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			ctx.PeerWrite()
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(ctx.Packet) {
				return ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite()
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func Sync(ctx *Context) (bool, error) {
	ctx.Packet = ctx.Packet.Reset(packets.TypeSync)
	if err := ctx.ServerWrite(); err != nil {
		return false, err
	}

	for {
		err := ctx.ServerRead()
		if err != nil {
			return false, err
		}

		switch ctx.Packet.Type() {
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
			ctx.PeerWrite()
		case packets.TypeCopyInResponse:
			if err = CopyIn(ctx); err != nil {
				return false, err
			}
			// why
			return false, nil
		case packets.TypeCopyOutResponse:
			if err = CopyOut(ctx); err != nil {
				return false, err
			}
		case packets.TypeReadyForQuery:
			var txState packets.ReadyForQuery
			if !txState.ReadFromPacket(ctx.Packet) {
				return false, ErrBadFormat
			}
			ctx.TxState = byte(txState)
			ctx.PeerWrite()
			return true, nil
		default:
			return false, ErrUnexpectedPacket
		}
	}
}

func EQP(ctx *Context) error {
	if err := ctx.ServerWrite(); err != nil {
		return err
	}

	for {
		if !ctx.PeerRead() {
			for {
				ok, err := Sync(ctx)
				if err != nil {
					return err
				}
				if ok {
					return nil
				}
			}
		}

		switch ctx.Packet.Type() {
		case packets.TypeSync:
			ok, err := Sync(ctx)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := ctx.ServerWrite(); err != nil {
				return err
			}
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}
	}
}

func Transaction(ctx *Context) error {
	if ctx.TxState == '\x00' {
		ctx.TxState = 'I'
	}
	for {
		switch ctx.Packet.Type() {
		case packets.TypeQuery:
			if err := Query(ctx); err != nil {
				return err
			}
		case packets.TypeFunctionCall:
			if err := FunctionCall(ctx); err != nil {
				return err
			}
		case packets.TypeSync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			rfq := packets.ReadyForQuery(ctx.TxState)
			ctx.Packet = rfq.IntoPacket(ctx.Packet)
			ctx.PeerWrite()
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := EQP(ctx); err != nil {
				return err
			}
		default:
			ctx.PeerFail(ErrUnexpectedPacket)
		}

		if ctx.TxState == 'I' {
			return nil
		}

		if !ctx.PeerRead() {
			// abort tx
			err := QueryString(ctx, "ABORT;")
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
