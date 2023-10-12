package backends

import (
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func copyIn(ctx *context) error {
	ctx.PeerWrite()

	for {
		if !ctx.PeerRead() {
			copyFail := packets.CopyFail("peer failed")
			ctx.Packet = &copyFail
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
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, ctx.Packet)
			if err != nil {
				return err
			}
			ctx.Packet = &p
			ctx.TxState = byte(p)
			ctx.PeerWrite()
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func queryString(ctx *context, q string) error {
	qq := packets.Query(q)
	ctx.Packet = &qq
	return query(ctx)
}

func QueryString(server, peer *fed.Conn, query string) (err, peerError error) {
	ctx := context{
		Server: server,
		Peer:   peer,
	}
	err = queryString(&ctx, query)
	peerError = ctx.PeerError
	return
}

func SetParameter(server, peer *fed.Conn, name strutil.CIString, value string) (err, peerError error) {
	var q strings.Builder
	escapedName := strutil.Escape(name.String(), '"')
	escapedValue := strutil.Escape(value, '\'')
	q.Grow(len(`SET "" = ''`) + len(escapedName) + len(escapedValue))
	q.WriteString(`SET "`)
	q.WriteString(escapedName)
	q.WriteString(`" = '`)
	q.WriteString(escapedValue)
	q.WriteString(`'`)

	return QueryString(
		server,
		peer,
		q.String(),
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
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, ctx.Packet)
			if err != nil {
				return err
			}
			ctx.Packet = &p
			ctx.TxState = byte(p)
			ctx.PeerWrite()
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}

func sync(ctx *context) (bool, error) {
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
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, ctx.Packet)
			if err != nil {
				return false, err
			}
			ctx.Packet = &p
			ctx.TxState = byte(p)
			ctx.PeerWrite()
			return true, nil
		default:
			return false, ErrUnexpectedPacket
		}
	}
}

func Sync(server, peer *fed.Conn) (err, peerErr error) {
	ctx := context{
		Server: server,
		Peer:   peer,
		Packet: &packets.Sync{},
	}
	_, err = sync(&ctx)
	peerErr = ctx.PeerError
	return
}

func eqp(ctx *context) error {
	if err := ctx.ServerWrite(); err != nil {
		return err
	}

	for {
		if !ctx.PeerRead() {
			for {
				ctx.Packet = &packets.Sync{}
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
			ctx.Packet = &rfq
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

func Transaction(server, peer *fed.Conn, initialPacket fed.Packet) (err, peerError error) {
	ctx := context{
		Server: server,
		Peer:   peer,
		Packet: initialPacket,
	}
	err = transaction(&ctx)
	peerError = ctx.PeerError
	return
}
