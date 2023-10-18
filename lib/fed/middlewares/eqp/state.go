package eqp

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/maps"
	"gfx.cafe/gfx/pggat/lib/util/ring"
)

type CloseVariant int

const (
	CloseVariantPreparedStatement CloseVariant = iota
	CloseVariantPortal
)

type Close struct {
	Variant CloseVariant
	Target  string
}

type State struct {
	preparedStatements map[string]*packets.Parse
	portals            map[string]*packets.Bind

	pendingPreparedStatements ring.Ring[*packets.Parse]
	pendingPortals            ring.Ring[*packets.Bind]
	pendingCloses             ring.Ring[Close]
}

// C2S is client to server packets
func (T *State) C2S(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeClose:
		return T.Close(packet)
	case packets.TypeParse:
		return T.Parse(packet)
	case packets.TypeBind:
		return T.Bind(packet)
	case packets.TypeQuery:
		T.Query()
		return packet, nil
	default:
		return packet, nil
	}
}

// S2C is server to client packets
func (T *State) S2C(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeCloseComplete:
		T.CloseComplete()
		return packet, nil
	case packets.TypeParseComplete:
		T.ParseComplete()
		return packet, nil
	case packets.TypeBindComplete:
		T.BindComplete()
		return packet, nil
	case packets.TypeCommandComplete:
		return T.CommandComplete(packet)
	case packets.TypeReadyForQuery:
		return T.ReadyForQuery(packet)
	default:
		return packet, nil
	}
}

// Close is a pending close. Execute on Close C->S
func (T *State) Close(packet fed.Packet) (fed.Packet, error) {
	var p packets.Close
	err := fed.ToConcrete(&p, packet)
	if err != nil {
		return nil, err
	}

	var variant CloseVariant
	switch p.Which {
	case 'S':
		variant = CloseVariantPreparedStatement
	case 'P':
		variant = CloseVariantPortal
	default:
		return nil, packets.ErrInvalidFormat
	}

	T.pendingCloses.PushBack(Close{
		Variant: variant,
		Target:  p.Name,
	})

	return &p, nil
}

// CloseComplete notifies that a close was successful. Execute on CloseComplete S->C
func (T *State) CloseComplete() {
	c, ok := T.pendingCloses.PopFront()
	if !ok {
		return
	}

	switch c.Variant {
	case CloseVariantPortal:
		delete(T.portals, c.Target)
	case CloseVariantPreparedStatement:
		delete(T.preparedStatements, c.Target)
	default:
		return
	}
}

// Parse is a pending prepared statement. Execute on Parse C->S
func (T *State) Parse(packet fed.Packet) (fed.Packet, error) {
	var p packets.Parse
	err := fed.ToConcrete(&p, packet)
	if err != nil {
		return nil, err
	}
	T.pendingPreparedStatements.PushBack(&p)
	return &p, nil
}

// ParseComplete notifies that a parse was successful. Execute on ParseComplete S->C
func (T *State) ParseComplete() {
	preparedStatement, ok := T.pendingPreparedStatements.PopFront()
	if !ok {
		return
	}

	if T.preparedStatements == nil {
		T.preparedStatements = make(map[string]*packets.Parse)
	}
	T.preparedStatements[preparedStatement.Destination] = preparedStatement
}

// Bind is a pending portal. Execute on Bind C->S
func (T *State) Bind(packet fed.Packet) (fed.Packet, error) {
	var p packets.Bind
	err := fed.ToConcrete(&p, packet)
	if err != nil {
		return nil, err
	}
	T.pendingPortals.PushBack(&p)
	return &p, nil
}

// BindComplete notifies that a bind was successful. Execute on BindComplete S->C
func (T *State) BindComplete() {
	portal, ok := T.pendingPortals.PopFront()
	if !ok {
		return
	}

	if T.portals == nil {
		T.portals = make(map[string]*packets.Bind)
	}
	T.portals[portal.Destination] = portal
}

// Query clobbers the unnamed portal and unnamed prepared statement. Execute on Query C->S
func (T *State) Query() {
	delete(T.portals, "")
	delete(T.preparedStatements, "")
}

// CommandComplete clobbers everything if DISCARD ALL | DEALLOCATE | CLOSE
func (T *State) CommandComplete(packet fed.Packet) (fed.Packet, error) {
	var p packets.CommandComplete
	err := fed.ToConcrete(&p, packet)
	if err != nil {
		return nil, err
	}

	if p == "DISCARD ALL" {
		maps.Clear(T.preparedStatements)
		maps.Clear(T.portals)
		T.pendingPreparedStatements.Clear()
		T.pendingPortals.Clear()
		T.pendingCloses.Clear()
	}

	return &p, nil
}

// ReadyForQuery clobbers portals if state == 'I' and pending. Execute on ReadyForQuery S->C
func (T *State) ReadyForQuery(packet fed.Packet) (fed.Packet, error) {
	var p packets.ReadyForQuery
	err := fed.ToConcrete(&p, packet)
	if err != nil {
		return nil, err
	}

	if p == 'I' {
		// clobber all portals
		for name := range T.portals {
			delete(T.portals, name)
		}
	}

	// all pending has failed
	T.pendingPreparedStatements.Clear()
	T.pendingPortals.Clear()
	T.pendingCloses.Clear()

	return &p, nil
}

func (T *State) Set(other *State) {
	maps.Clear(T.preparedStatements)
	maps.Clear(T.portals)

	T.pendingPreparedStatements.Clear()
	T.pendingPortals.Clear()
	T.pendingCloses.Clear()

	if T.preparedStatements == nil {
		T.preparedStatements = make(map[string]*packets.Parse)
	}
	if T.portals == nil {
		T.portals = make(map[string]*packets.Bind)
	}

	for k, v := range other.preparedStatements {
		T.preparedStatements[k] = v
	}
	for k, v := range other.portals {
		T.portals[k] = v
	}
	for i := 0; i < other.pendingPreparedStatements.Length(); i++ {
		T.pendingPreparedStatements.PushBack(other.pendingPreparedStatements.Get(i))
	}
	for i := 0; i < other.pendingPortals.Length(); i++ {
		T.pendingPortals.PushBack(other.pendingPortals.Get(i))
	}
	for i := 0; i < other.pendingCloses.Length(); i++ {
		T.pendingCloses.PushBack(other.pendingCloses.Get(i))
	}
}
