package eqp

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
	"pggat2/lib/util/decorator"
	"pggat2/lib/util/ring"
)

type Creator struct {
	noCopy decorator.NoCopy

	preparedStatements        map[string]PreparedStatement
	portals                   map[string]Portal
	pendingPreparedStatements ring.Ring[string]
	pendingPortals            ring.Ring[string]
	inner                     pnet.ReadWriter
}

func MakeCreator(inner pnet.ReadWriter) Creator {
	return Creator{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
		inner:              inner,
	}
}

func NewCreator(inner pnet.ReadWriter) *Creator {
	c := MakeCreator(inner)
	return &c
}

func (T *Creator) Read() (packet.In, error) {
	for {
		in, err := T.inner.Read()
		if err != nil {
			return packet.In{}, err
		}
		switch in.Type() {
		case packet.Query:
			// clobber unnamed portal and unnamed prepared statement
			delete(T.preparedStatements, "")
			delete(T.portals, "")
			return in, nil
		case packet.Parse:
			destination, query, parameterDataTypes, ok := packets.ReadParse(in)
			if !ok {
				return packet.In{}, ErrBadPacketFormat
			}
			if destination != "" {
				if _, ok = T.preparedStatements[destination]; ok {
					return packet.In{}, ErrPreparedStatementExists
				}
			}
			T.preparedStatements[destination] = PreparedStatement{
				Query:              query,
				ParameterDataTypes: parameterDataTypes,
			}
			T.pendingPreparedStatements.PushBack(destination)

			// send parse complete
			out := T.inner.Write()
			out.Type(packet.ParseComplete)
			err = T.inner.Send(out.Finish())
			if err != nil {
				return packet.In{}, err
			}
		case packet.Bind:
			destination, source, parameterFormatCodes, parameterValues, resultFormatCodes, ok := packets.ReadBind(in)
			if !ok {
				return packet.In{}, ErrBadPacketFormat
			}
			if destination != "" {
				if _, ok = T.portals[destination]; ok {
					return packet.In{}, ErrPortalExists
				}
			}
			T.portals[destination] = Portal{
				Source:               source,
				ParameterFormatCodes: parameterFormatCodes,
				ParameterValues:      parameterValues,
				ResultFormatCodes:    resultFormatCodes,
			}
			T.pendingPortals.PushBack(destination)

			// send bind complete
			out := T.inner.Write()
			out.Type(packet.BindComplete)
			err = T.inner.Send(out.Finish())
			if err != nil {
				return packet.In{}, err
			}
		case packet.Close:
			which, target, ok := packets.ReadClose(in)
			if !ok {
				return packet.In{}, ErrBadPacketFormat
			}
			switch which {
			case 'S':
				delete(T.preparedStatements, target)
			case 'P':
				delete(T.portals, target)
			default:
				return packet.In{}, ErrBadPacketFormat
			}
		default:
			return in, nil
		}
	}
}

func (T *Creator) ReadUntyped() (packet.In, error) {
	return T.inner.ReadUntyped()
}

func (T *Creator) Write() packet.Out {
	return T.inner.Write()
}

func (T *Creator) WriteByte(b byte) error {
	return T.inner.WriteByte(b)
}

func (T *Creator) Send(typ packet.Type, payload []byte) error {
	switch typ {
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
	return T.inner.Send(typ, payload)
}

var _ pnet.ReadWriter = (*Creator)(nil)
