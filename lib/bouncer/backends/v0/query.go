package backends

import (
	"context"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func copyIn(ctx context.Context, binding *serverToPeerBinding) error {
	binding.PeerWrite(ctx)

	for {
		if !binding.PeerRead(ctx) {
			copyFail := packets.CopyFail("peer failed")
			binding.Packet = &copyFail
			return binding.ServerWrite(ctx)
		}

		switch binding.Packet.Type() {
		case packets.TypeCopyData:
			if err := binding.ServerWrite(ctx); err != nil {
				return err
			}
		case packets.TypeCopyDone, packets.TypeCopyFail:
			return binding.ServerWrite(ctx)
		default:
			binding.PeerFail(binding.ErrUnexpectedPacket())
		}
	}
}

func copyOut(ctx context.Context, binding *serverToPeerBinding) error {
	binding.PeerWrite(ctx)

	for {
		err := binding.ServerRead(ctx)
		if err != nil {
			return err
		}

		switch binding.Packet.Type() {
		case packets.TypeCopyData,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			binding.PeerWrite(ctx)
		case packets.TypeCopyDone, packets.TypeErrorResponse:
			binding.PeerWrite(ctx)
			return nil
		default:
			return binding.ErrUnexpectedPacket()
		}
	}
}

func query(ctx context.Context, binding *serverToPeerBinding) error {
	if err := binding.ServerWrite(ctx); err != nil {
		return err
	}

	for {
		err := binding.ServerRead(ctx)
		if err != nil {
			return err
		}

		switch binding.Packet.Type() {
		case packets.TypeCommandComplete,
			packets.TypeRowDescription,
			packets.TypeDataRow,
			packets.TypeEmptyQueryResponse,
			packets.TypeErrorResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			binding.PeerWrite(ctx)
		case packets.TypeCopyInResponse:
			if err = copyIn(ctx, binding); err != nil {
				return err
			}
		case packets.TypeCopyOutResponse:
			if err = copyOut(ctx, binding); err != nil {
				return err
			}
		case packets.TypeReadyForQuery:
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, binding.Packet)
			if err != nil {
				return err
			}
			binding.Packet = &p
			binding.TxState = byte(p)
			binding.PeerWrite(ctx)
			return nil
		default:
			return binding.ErrUnexpectedPacket()
		}
	}
}

func queryString(ctx context.Context, binding *serverToPeerBinding, q string) error {
	qq := packets.Query(q)
	binding.Packet = &qq
	return query(ctx, binding)
}

func QueryString(ctx context.Context, server, peer *fed.Conn, query string) (err, peerError error) {
	binding := serverToPeerBinding{
		Server: server,
		Peer:   peer,
	}
	err = queryString(ctx, &binding, query)
	peerError = binding.PeerError
	return
}

func SetParameter(ctx context.Context, server, peer *fed.Conn, name strutil.CIString, value string) (err, peerError error) {
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
		ctx,
		server,
		peer,
		q.String(),
	)
}

func functionCall(ctx context.Context, binding *serverToPeerBinding) error {
	if err := binding.ServerWrite(ctx); err != nil {
		return err
	}

	for {
		err := binding.ServerRead(ctx)
		if err != nil {
			return err
		}

		switch binding.Packet.Type() {
		case packets.TypeErrorResponse,
			packets.TypeFunctionCallResponse,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			binding.PeerWrite(ctx)
		case packets.TypeReadyForQuery:
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, binding.Packet)
			if err != nil {
				return err
			}
			binding.Packet = &p
			binding.TxState = byte(p)
			binding.PeerWrite(ctx)
			return nil
		default:
			return binding.ErrUnexpectedPacket()
		}
	}
}

func sync(ctx context.Context, binding *serverToPeerBinding) (bool, error) {
	if err := binding.ServerWrite(ctx); err != nil {
		return false, err
	}

	for {
		err := binding.ServerRead(ctx)
		if err != nil {
			return false, err
		}

		switch binding.Packet.Type() {
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
			binding.PeerWrite(ctx)
		case packets.TypeCopyInResponse:
			if err = copyIn(ctx, binding); err != nil {
				return false, err
			}
			// why
			return false, nil
		case packets.TypeCopyOutResponse:
			if err = copyOut(ctx, binding); err != nil {
				return false, err
			}
		case packets.TypeReadyForQuery:
			var p packets.ReadyForQuery
			err = fed.ToConcrete(&p, binding.Packet)
			if err != nil {
				return false, err
			}
			binding.Packet = &p
			binding.TxState = byte(p)
			binding.PeerWrite(ctx)
			return true, nil
		default:
			return false, binding.ErrUnexpectedPacket()
		}
	}
}

func Sync(ctx context.Context, server, peer *fed.Conn) (err, peerErr error) {
	binding := serverToPeerBinding{
		Server: server,
		Peer:   peer,
		Packet: &packets.Sync{},
	}
	_, err = sync(ctx, &binding)
	peerErr = binding.PeerError
	return
}

func eqp(ctx context.Context, binding *serverToPeerBinding) error {
	if err := binding.ServerWrite(ctx); err != nil {
		return err
	}

	for {
		if !binding.PeerRead(ctx) {
			for {
				binding.Packet = &packets.Sync{}
				ok, err := sync(ctx, binding)
				if err != nil {
					return err
				}
				if ok {
					return nil
				}
			}
		}

		switch binding.Packet.Type() {
		case packets.TypeSync:
			ok, err := sync(ctx, binding)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := binding.ServerWrite(ctx); err != nil {
				return err
			}
		default:
			binding.PeerFail(binding.ErrUnexpectedPacket())
		}
	}
}

func transaction(ctx context.Context, binding *serverToPeerBinding) error {
	for {
		switch binding.Packet.Type() {
		case packets.TypeQuery:
			if err := query(ctx, binding); err != nil {
				return err
			}
		case packets.TypeFunctionCall:
			if err := functionCall(ctx, binding); err != nil {
				return err
			}
		case packets.TypeSync:
			// phony sync call, we can just reply with a fake ReadyForQuery(TxState)
			rfq := packets.ReadyForQuery(binding.TxState)
			binding.Packet = &rfq
			binding.PeerWrite(ctx)
		case packets.TypeParse, packets.TypeBind, packets.TypeClose, packets.TypeDescribe, packets.TypeExecute, packets.TypeFlush:
			if err := eqp(ctx, binding); err != nil {
				return err
			}
		default:
			binding.PeerFail(binding.ErrUnexpectedPacket())
		}

		if binding.TxState == 'I' {
			return nil
		}

		if !binding.PeerRead(ctx) {
			// abort tx
			err := queryString(ctx, binding, "ABORT;")
			if err != nil {
				return err
			}

			if binding.TxState != 'I' {
				return ErrExpectedIdle
			}
			return nil
		}
	}
}

func Transaction(ctx context.Context, server, peer *fed.Conn, initialPacket fed.Packet) (err, peerError error) {
	pgState := serverToPeerBinding{
		Server: server,
		Peer:   peer,
		Packet: initialPacket,
	}
	err = transaction(ctx, &pgState)
	peerError = pgState.PeerError
	return
}
