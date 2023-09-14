package eqp

import (
	"hash/maphash"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/util/ring"
)

var seed = maphash.MakeSeed()

type PreparedStatement struct {
	Packet fed.Packet
	Target string
	Hash   uint64
}

func MakePreparedStatement(packet fed.Packet) PreparedStatement {
	if packet.Type() != packets.TypeParse {
		panic("unreachable")
	}

	var res PreparedStatement
	packet.ReadString(&res.Target)
	res.Packet = packet
	res.Hash = maphash.Bytes(seed, packet.Payload())

	return res
}

type Portal struct {
	Packet fed.Packet
	Target string
}

func MakePortal(packet fed.Packet) Portal {
	if packet.Type() != packets.TypeBind {
		panic("unreachable")
	}

	var res Portal
	packet.ReadString(&res.Target)
	res.Packet = packet

	return res
}

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
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal

	pendingPreparedStatements ring.Ring[PreparedStatement]
	pendingPortals            ring.Ring[Portal]
	pendingCloses             ring.Ring[Close]
}

// C2S is client to server packets
func (T *State) C2S(packet fed.Packet) {
	switch packet.Type() {
	case packets.TypeClose:
		T.Close(packet)
	case packets.TypeParse:
		T.Parse(packet)
	case packets.TypeBind:
		T.Bind(packet)
	case packets.TypeQuery:
		T.Query()
	}
}

// S2C is server to client packets
func (T *State) S2C(packet fed.Packet) {
	switch packet.Type() {
	case packets.TypeCloseComplete:
		T.CloseComplete()
	case packets.TypeParseComplete:
		T.ParseComplete()
	case packets.TypeBindComplete:
		T.BindComplete()
	case packets.TypeReadyForQuery:
		T.ReadyForQuery(packet)
	}
}

// Close is a pending close. Execute on Close C->S
func (T *State) Close(packet fed.Packet) {
	var which byte
	p := packet.ReadUint8(&which)
	var target string
	p.ReadString(&target)

	var variant CloseVariant
	switch which {
	case 'S':
		variant = CloseVariantPreparedStatement
	case 'P':
		variant = CloseVariantPortal
	default:
		return
	}

	T.pendingCloses.PushBack(Close{
		Variant: variant,
		Target:  target,
	})
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
func (T *State) Parse(packet fed.Packet) {
	preparedStatement := MakePreparedStatement(packet)
	T.pendingPreparedStatements.PushBack(preparedStatement)
}

// ParseComplete notifies that a parse was successful. Execute on ParseComplete S->C
func (T *State) ParseComplete() {
	preparedStatement, ok := T.pendingPreparedStatements.PopFront()
	if !ok {
		return
	}

	if T.preparedStatements == nil {
		T.preparedStatements = make(map[string]PreparedStatement)
	}
	T.preparedStatements[preparedStatement.Target] = preparedStatement
}

// Bind is a pending portal. Execute on Bind C->S
func (T *State) Bind(packet fed.Packet) {
	portal := MakePortal(packet)
	T.pendingPortals.PushBack(portal)
}

// BindComplete notifies that a bind was successful. Execute on BindComplete S->C
func (T *State) BindComplete() {
	portal, ok := T.pendingPortals.PopFront()
	if !ok {
		return
	}

	if T.portals == nil {
		T.portals = make(map[string]Portal)
	}
	T.portals[portal.Target] = portal
}

// Query clobbers the unnamed portal and unnamed prepared statement. Execute on Query C->S
func (T *State) Query() {
	delete(T.portals, "")
	delete(T.preparedStatements, "")
}

// ReadyForQuery clobbers portals if state == 'I' and pending. Execute on ReadyForQuery S->C
func (T *State) ReadyForQuery(packet fed.Packet) {
	var state byte
	packet.ReadUint8(&state)

	if state == 'I' {
		// clobber all portals
		for name := range T.portals {
			delete(T.portals, name)
		}
	}

	// all pending has failed
	T.pendingPreparedStatements.Clear()
	T.pendingPortals.Clear()
	T.pendingCloses.Clear()
}
