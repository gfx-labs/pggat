package eqp2

import (
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/util/ring"
)

type PreparedStatement struct {
	Packet fed.Packet
	Target string
}

func MakePreparedStatement(packet fed.Packet) PreparedStatement {
	if packet.Type() != packets.TypeParse {
		panic("unreachable")
	}
}

type Portal struct {
	Packet fed.Packet
	Target string
}

func MakePortal(packet fed.Packet) Portal {
	if packet.Type() != packets.TypeBind {
		panic("unreachable")
	}
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

type Sync struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal

	pendingPreparedStatements ring.Ring[PreparedStatement]
	pendingPortals            ring.Ring[Portal]
	pendingCloses             ring.Ring[Close]
}

// Close is a pending close. Execute on Close C->S
func (T *Sync) Close(variant CloseVariant, target string) {
	T.pendingCloses.PushBack(Close{
		Variant: variant,
		Target:  target,
	})
}

// CloseComplete notifies that a close was successful. Execute on CloseComplete S->C
func (T *Sync) CloseComplete() {
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
func (T *Sync) Parse(packet fed.Packet) {
	preparedStatement := MakePreparedStatement(packet)
	T.pendingPreparedStatements.PushBack(preparedStatement)
}

// ParseComplete notifies that a parse was successful. Execute on ParseComplete S->C
func (T *Sync) ParseComplete() {
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
func (T *Sync) Bind(packet fed.Packet) {
	portal := MakePortal(packet)
	T.pendingPortals.PushBack(portal)
}

// BindComplete notifies that a bind was successful. Execute on BindComplete S->C
func (T *Sync) BindComplete() {
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
func (T *Sync) Query() {
	delete(T.portals, "")
	delete(T.preparedStatements, "")
}

// ReadyForQuery clobbers portals if state == 'I' and pending. Execute on ReadyForQuery S->C
func (T *Sync) ReadyForQuery(state byte) {
	if state == 'I' {
		// clobber all portals
		for name := range T.portals {
			delete(T.portals, name)
		}
	}

	// all pending has failed
	for _, ok := T.pendingPreparedStatements.PopBack(); ok; _, ok = T.pendingPortals.PopBack() {
	}
	for _, ok := T.pendingPortals.PopBack(); ok; _, ok = T.pendingPortals.PopBack() {
	}
	for _, ok := T.pendingCloses.PopBack(); ok; _, ok = T.pendingPortals.PopBack() {
	}
}
