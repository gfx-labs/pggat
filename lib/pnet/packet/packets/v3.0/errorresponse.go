package packets

import (
	"pggat2/lib/perror"
	"pggat2/lib/pnet/packet"
)

func ReadErrorResponse(in packet.In) (perror.Error, bool) {
	in.Reset()
	if in.Type() != packet.ErrorResponse {
		return nil, false
	}

	var severity perror.Severity
	var code perror.Code
	var message string
	var extra []perror.ExtraField

	for {
		typ, ok := in.Uint8()
		if !ok {
			return nil, false
		}

		if typ == 0 {
			break
		}

		value, ok := in.String()
		if !ok {
			return nil, false
		}

		switch typ {
		case 'S':
			severity = perror.Severity(value)
		case 'C':
			code = perror.Code(value)
		case 'M':
			message = value
		default:
			extra = append(extra, perror.ExtraField{
				Type:  perror.Extra(typ),
				Value: value,
			})
		}
	}

	return perror.New(
		severity,
		code,
		message,
		extra...,
	), true
}

func WriteErrorResponse(out packet.Out, err perror.Error) {
	out.Reset()
	out.Type(packet.ErrorResponse)

	out.Uint8('S')
	out.String(string(err.Severity()))

	out.Uint8('C')
	out.String(string(err.Code()))

	out.Uint8('M')
	out.String(err.Message())

	for _, field := range err.Extra() {
		out.Uint8(uint8(field.Type))
		out.String(field.Value)
	}

	out.Uint8(0)
}
