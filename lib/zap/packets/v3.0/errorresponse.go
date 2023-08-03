package packets

import (
	"pggat2/lib/perror"
	"pggat2/lib/zap"
)

func ReadErrorResponse(in zap.ReadablePacket) (perror.Error, bool) {
	if in.ReadType() != ErrorResponse {
		return nil, false
	}

	var severity perror.Severity
	var code perror.Code
	var message string
	var extra []perror.ExtraField

	for {
		typ, ok := in.ReadUint8()
		if !ok {
			return nil, false
		}

		if typ == 0 {
			break
		}

		value, ok := in.ReadString()
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

func WriteErrorResponse(out *zap.Packet, err perror.Error) {
	out.WriteType(ErrorResponse)

	out.WriteUint8('S')
	out.WriteString(string(err.Severity()))

	out.WriteUint8('C')
	out.WriteString(string(err.Code()))

	out.WriteUint8('M')
	out.WriteString(err.Message())

	for _, field := range err.Extra() {
		out.WriteUint8(uint8(field.Type))
		out.WriteString(field.Value)
	}

	out.WriteUint8(0)
}
