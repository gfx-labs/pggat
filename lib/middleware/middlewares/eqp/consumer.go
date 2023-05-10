package eqp

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
)

type Consumer struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal
	inner              pnet.ReadWriteSender
}

func MakeConsumer(inner pnet.ReadWriteSender) Consumer {
	return Consumer{
		preparedStatements: make(map[string]PreparedStatement),
		portals:            make(map[string]Portal),
		inner:              inner,
	}
}

func (T Consumer) Read() (packet.In, error) {
	return T.inner.Read()
}

func (T Consumer) ReadUntyped() (packet.In, error) {
	return T.inner.ReadUntyped()
}

func (T Consumer) Write() packet.Out {
	return T.inner.Write().WithSender(T)
}

func (T Consumer) WriteByte(b byte) error {
	return T.inner.WriteByte(b)
}

func (T Consumer) Send(typ packet.Type, bytes []byte) error {
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

var _ pnet.ReadWriteSender = Consumer{}
