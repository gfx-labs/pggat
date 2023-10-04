package backends

import (
	"fmt"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func copyIn(ctx *context) error {
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

func copyOut(ctx *context) error {
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

func query(ctx *context) error {
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
			if err = copyIn(ctx); err != nil {
				return err
			}
		case packets.TypeCopyOutResponse:
			if err = copyOut(ctx); err != nil {
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

func queryString(ctx *context, q string) error {
	qq := packets.Query(q)
	ctx.Packet = qq.IntoPacket(ctx.Packet)
	return query(ctx)
}

func QueryString(server, peer *fed.Conn, buffer fed.Packet, query string) (err, peerError error, packet fed.Packet) {
	ctx := context{
		Server: server,
		Peer:   peer,
		Packet: buffer,
	}
	err = queryString(&ctx, query)
	peerError = ctx.PeerError
	packet = ctx.Packet
	return
}

func SetParameter(server, peer *fed.Conn, buffer fed.Packet, name strutil.CIString, value string) (err, peerError error, packet fed.Packet) {
	return QueryString(
		server,
		peer,
		buffer,
		fmt.Sprintf(`SET "%s" = '%s'`, strutil.Escape(name.String(), '"'), strutil.Escape(value, '\'')),
	)
}

func functionCall(ctx *context) error {
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

func sync(ctx *context) (bool, error) {
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
			if err = copyIn(ctx); err != nil {
				return false, err
			}
			// why
			return false, nil
		case packets.TypeCopyOutResponse:
			if err = copyOut(ctx); err != nil {
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

func Sync(server, peer *fed.Conn, buffer fed.Packet) (err, peerErr error, packet fed.Packet) {
	ctx := context{
		Server: server,
		Peer:   peer,
		Packet: buffer,
	}
	_, err = sync(&ctx)
	peerErr = ctx.PeerError
	packet = ctx.Packet
	return
}

func eqp(ctx *context) error {
	if err := ctx.ServerWrite(); err != nil {
		return err
	}

	for {
		if !ctx.PeerRead() {
			for {
				ok, err := sync(ctx)
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
			ok, err := sync(ctx)
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

func transaction(ctx *context) error {
	for {
		switch ctx.Packet.Type() {
		case packets.TypeQuery:
			if err := query(ctx); err != nil {
				return err
			}
		case packets.TypeFunctionCall:
			if err := functionCall(ctx); err != nil {
				return err
			}
		case packets.TypeSync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			rfq := packets.ReadyForQuery(ctx.TxState)
			ctx.Packet = rfq.IntoPacket(ctx.Packet)
			ctx.PeerWrite()
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := eqp(ctx); err != nil {
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
			err := queryString(ctx, "ABORT;")
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

func Transaction(server, peer *fed.Conn, initialPacket fed.Packet) (err, peerError error, packet fed.Packet) {
	ctx := context{
		Server: server,
		Peer:   peer,
		Packet: initialPacket,
	}
	err = transaction(&ctx)
	peerError = ctx.PeerError
	packet = ctx.Packet
	return
}
