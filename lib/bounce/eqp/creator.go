package eqp

import (
	"errors"

	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
)

type Creator struct {
	preparedStatements map[string]PreparedStatement
	portals            map[string]Portal
	inner              pnet.ReadWriteSender
}

func (T Creator) Read() (packet.In, error) {
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
				return packet.In{}, errors.New("bad packet format")
			}
			T.preparedStatements[destination] = PreparedStatement{
				Query:              query,
				ParameterDataTypes: parameterDataTypes,
			}
		case packet.Bind:
			destination, source, parameterFormatCodes, parameterValues, resultFormatCodes, ok := packets.ReadBind(in)
			if !ok {
				return packet.In{}, errors.New("bad packet format")
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
				return packet.In{}, errors.New("bad packet format")
			}
			switch which {
			case 'S':
				delete(T.preparedStatements, target)
			case 'P':
				delete(T.portals, target)
			default:
				return packet.In{}, errors.New("unknown close target")
			}
		default:
			return in, nil
		}
	}
}

func (T Creator) ReadUntyped() (packet.In, error) {
	return T.inner.ReadUntyped()
}

func (T Creator) Write() packet.Out {
	return T.inner.Write()
}

func (T Creator) Send(typ packet.Type, payload []byte) error {
	return T.inner.Send(typ, payload)
}

var _ pnet.ReadWriteSender = Creator{}
