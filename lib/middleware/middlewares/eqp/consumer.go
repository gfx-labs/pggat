package eqp

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
	"pggat2/lib/util/decorator"
	"pggat2/lib/util/ring"
)

type Consumer struct {
	noCopy decorator.NoCopy

	preparedStatements        map[string]PreparedStatement
	portals                   map[string]Portal
	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]
	inner                     pnet.ReadWriter
}

func MakeConsumer(inner pnet.ReadWriter) Consumer {
	return Consumer{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
		inner:              inner,
	}
}

func NewConsumer(inner pnet.ReadWriter) *Consumer {
	c := MakeConsumer(inner)
	return &c
}

func (T *Consumer) Read() (packet.In, error) {
	in, err := T.inner.Read()
	if err != nil {
		return packet.In{}, err
	}
	switch in.Type() {
	case packet.ParseComplete:
		T.pendingPreparedStatements.PopFront()
	case packet.BindComplete:
		T.pendingPortals.PopFront()
	case packet.ReadyForQuery:
		// remove all pending, they were not added.
		for pending, ok := T.pendingPreparedStatements.PopFront(); ok; pending, ok = T.pendingPreparedStatements.PopFront() {
			delete(T.preparedStatements, pending)
		}
		for pending, ok := T.pendingPortals.PopFront(); ok; pending, ok = T.pendingPortals.PopFront() {
			delete(T.portals, pending)
		}
	}
	return in, nil
}

func (T *Consumer) ReadUntyped() (packet.In, error) {
	return T.inner.ReadUntyped()
}

func (T *Consumer) Write() packet.Out {
	return T.inner.Write()
}

func (T *Consumer) WriteByte(b byte) error {
	return T.inner.WriteByte(b)
}

func (T *Consumer) Send(typ packet.Type, bytes []byte) error {
	buf := packet.MakeInBuf(typ, bytes)
	in := packet.MakeIn(&buf)
	switch typ {
	case packet.Query:
		// clobber unnamed portal and unnamed prepared statement
		delete(T.preparedStatements, "")
		delete(T.portals, "")
	case packet.Parse:
		destination, query, parameterDataTypes, ok := packets.ReadParse(in)
		if !ok {
			return ErrBadPacketFormat
		}
		if destination != "" {
			if _, ok = T.preparedStatements[destination]; ok {
				return ErrPreparedStatementExists
			}
		}
		T.preparedStatements[destination] = PreparedStatement{
			Query:              query,
			ParameterDataTypes: parameterDataTypes,
		}
		T.pendingPreparedStatements.PushBack(destination)
	case packet.Bind:
		destination, source, parameterFormatCodes, parameterValues, resultFormatCodes, ok := packets.ReadBind(in)
		if !ok {
			return ErrBadPacketFormat
		}
		if destination != "" {
			if _, ok = T.portals[destination]; ok {
				return ErrPortalExists
			}
		}
		T.portals[destination] = Portal{
			Source:               source,
			ParameterFormatCodes: parameterFormatCodes,
			ParameterValues:      parameterValues,
			ResultFormatCodes:    resultFormatCodes,
		}
		T.pendingPortals.PushBack(destination)
	case packet.Close:
		which, target, ok := packets.ReadClose(in)
		if !ok {
			return ErrBadPacketFormat
		}
		switch which {
		case 'S':
			delete(T.preparedStatements, target)
		case 'P':
			delete(T.portals, target)
		default:
			return ErrUnknownCloseTarget
		}
	}
	return T.inner.Send(typ, bytes)
}

var _ pnet.ReadWriter = (*Consumer)(nil)
