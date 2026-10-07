package eqp

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

type Server struct {
	state State

	// lazy holds the paired client's prepared statements that this server lacks. Each is parsed
	// just before the first Bind or statement Describe that names it, so a statement that no
	// longer parses fails only the request that uses it, with its own error.
	lazy map[string]*packets.Parse
	// deferred is the request held back while its injected Parse is written.
	deferred fed.Packet
	// syncing is set while SyncMiddleware reads the responses to its own requests.
	syncing bool
}

func NewServer() *Server {
	return new(Server)
}

func (T *Server) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Server) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeParseComplete:
		pending, ok := T.state.ParseComplete()
		if ok {
			delete(T.lazy, pending.parse.Destination)
		}
		if ok && pending.injected {
			// the client did not send this Parse
			return nil, nil
		}
		return packet, nil
	case packets.TypeCloseComplete:
		// pairing closes its own statements, so only a client Close removes a lazy definition
		if !T.syncing && T.state.pendingCloses.Length() > 0 {
			if c := T.state.pendingCloses.Get(0); c.Variant == CloseVariantPreparedStatement {
				delete(T.lazy, c.Target)
			}
		}
		return T.state.S2C(packet)
	case packets.TypeCommandComplete:
		p, err := T.state.CommandComplete(packet)
		if err != nil {
			return nil, err
		}
		if cc := *p.(*packets.CommandComplete); cc == "DEALLOCATE ALL" || cc == "DISCARD ALL" {
			clear(T.lazy)
		}
		return p, nil
	default:
		return T.state.S2C(packet)
	}
}

func (T *Server) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	if T.deferred != nil {
		deferred := T.deferred
		T.deferred = nil
		if packet == deferred {
			// tracked when it was deferred
			return packet, nil
		}
	}

	var statement string
	switch packet.Type() {
	case packets.TypeDescribe:
		var p packets.Describe
		if err := fed.ToConcrete(&p, packet); err != nil {
			return nil, err
		}
		packet = &p
		if p.Which != 'S' {
			return packet, nil
		}
		statement = p.Name
	case packets.TypeBind:
		p, err := T.state.Bind(packet)
		if err != nil {
			return nil, err
		}
		packet = p
		statement = p.(*packets.Bind).Source
	case packets.TypeParse:
		var p packets.Parse
		if err := fed.ToConcrete(&p, packet); err != nil {
			return nil, err
		}
		if p.Destination == "" {
			// PostgreSQL drops the unnamed statement before parsing its replacement
			delete(T.lazy, "")
			return T.state.Parse(&p)
		}
		// the client's session holds this name, so the server must hold it too and report 42P05
		parse := T.injection(p.Destination)
		if parse == nil {
			return T.state.Parse(&p)
		}
		// track in wire order
		T.state.inject(parse)
		if _, err := T.state.Parse(&p); err != nil {
			return nil, err
		}
		T.deferred = &p
		return parse, nil
	case packets.TypeClose:
		// lazy is updated on CloseComplete, because a Close after an error is skipped
		return T.state.Close(packet)
	case packets.TypeQuery:
		T.state.Query()
		delete(T.lazy, "")
		return packet, nil
	default:
		return packet, nil
	}

	parse := T.injection(statement)
	if parse == nil {
		return packet, nil
	}
	T.state.inject(parse)
	T.deferred = packet
	return parse, nil
}

// injection returns the client's Parse for statement if the server must receive it first.
func (T *Server) injection(statement string) *packets.Parse {
	parse, ok := T.lazy[statement]
	if !ok || T.state.pendingParse(statement) || T.state.pendingClose(statement) {
		return nil
	}
	if current, ok := T.state.preparedStatements[statement]; ok && preparedStatementsEqual(current, parse) {
		delete(T.lazy, statement)
		return nil
	}
	return parse
}

func (T *Server) PostWrite(ctx context.Context) (fed.Packet, error) {
	return T.deferred, nil
}

var _ fed.Middleware = (*Server)(nil)
