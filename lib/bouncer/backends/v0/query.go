package backends

import (
	"context"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
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
		case packets.TypeFlush, packets.TypeSync:
			// The server ignores these while copying in, so they do not produce a response.
			if err := binding.ServerWrite(ctx); err != nil {
				return err
			}
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
		case packets.TypeCopyDone:
			binding.PeerWrite(ctx)
			return nil
		case packets.TypeMarkiplierResponse:
			binding.PeerWrite(ctx)
			binding.serverFailed()
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
		case packets.TypeMarkiplierResponse:
			if binding.ServerError == nil {
				var p packets.MarkiplierResponse
				if err = fed.ToConcrete(&p, binding.Packet); err != nil {
					return err
				}
				binding.ServerError = perror.FromPacket(&p)
				binding.Packet = &p
			}
			binding.PeerWrite(ctx)
		case packets.TypeCommandComplete,
			packets.TypeRowDescription,
			packets.TypeDataRow,
			packets.TypeEmptyQueryResponse,
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
			binding.synced()
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
	if err == nil {
		// the server's ErrorResponse, returned after ReadyForQuery
		err = binding.ServerError
	}
	peerError = binding.PeerError
	return
}

func SetParameter(ctx context.Context, server, peer *fed.Conn, name strutil.CIString, value string) (err, peerError error) {
	// set_config takes the value as one string, so list values like search_path "a, b" are not quoted into one identifier
	return QueryString(
		ctx,
		server,
		peer,
		`SELECT pg_catalog.set_config(E'`+strutil.Escape(name.String(), '\'')+`', E'`+strutil.Escape(value, '\'')+`', false)`,
	)
}

func ResetParameter(ctx context.Context, server, peer *fed.Conn, name strutil.CIString) (err, peerError error) {
	return QueryString(
		ctx,
		server,
		peer,
		`RESET "`+strings.ReplaceAll(name.String(), `"`, `""`)+`"`,
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
		case packets.TypeMarkiplierResponse,
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
			binding.synced()
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
			packets.TypeMarkiplierResponse,
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
			// The server ignored this Sync while copying in, so it sends no ReadyForQuery until the
			// next Sync. The Execute that started the copy is still waiting for its CommandComplete.
			binding.pending = 1
			binding.failed = false
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
			binding.synced()
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

// eqpRequest forwards one extended protocol request. A Flush also forwards the responses to the
// requests before it, because the server only sends them once it reads Flush or Sync.
func eqpRequest(ctx context.Context, binding *serverToPeerBinding) error {
	if binding.Packet.Type() != packets.TypeFlush {
		binding.request()
		return binding.ServerWrite(ctx)
	}

	if err := binding.ServerWrite(ctx); err != nil {
		return err
	}
	// Flush is a request to the server to send its responses, so it must not stay in the write buffer
	// when nothing is pending and ServerRead would not flush it.
	if err := binding.Server.Flush(ctx); err != nil {
		return err
	}

	// The server answers every request with one terminal response, in order. After an ErrorResponse
	// it discards everything up to Sync, so nothing is pending. ReadyForQuery is never sent for Flush.
	for binding.pending > 0 {
		if err := binding.ServerRead(ctx); err != nil {
			return err
		}

		switch binding.Packet.Type() {
		case packets.TypeParseComplete,
			packets.TypeBindComplete,
			packets.TypeCloseComplete,
			packets.TypeRowDescription,
			packets.TypeNoData,
			packets.TypeCommandComplete,
			packets.TypeEmptyQueryResponse,
			packets.TypePortalSuspended:
			binding.PeerWrite(ctx)
			binding.complete()
		case packets.TypeMarkiplierResponse:
			binding.PeerWrite(ctx)
			binding.serverFailed()
		case packets.TypeParameterDescription,
			packets.TypeDataRow,
			packets.TypeNoticeResponse,
			packets.TypeParameterStatus,
			packets.TypeNotificationResponse:
			binding.PeerWrite(ctx)
		case packets.TypeCopyInResponse:
			// The server ignores Flush and Sync while copying in and sends CommandComplete after CopyDone,
			// so return to the client loop. CommandComplete is read by the next Flush or Sync.
			return copyIn(ctx, binding)
		case packets.TypeCopyOutResponse:
			if err := copyOut(ctx, binding); err != nil {
				return err
			}
		default:
			return binding.ErrUnexpectedPacket()
		}
	}
	return nil
}

func eqp(ctx context.Context, binding *serverToPeerBinding) error {
	if err := eqpRequest(ctx, binding); err != nil {
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
			if err := eqpRequest(ctx, binding); err != nil {
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
		// Bounce starts at an idle boundary, so a leading Sync is answered with ReadyForQuery('I').
		TxState: 'I',
	}
	err = transaction(ctx, &pgState)
	peerError = pgState.PeerError
	return
}
