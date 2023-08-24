package packets

import (
	"pggat2/lib/perror"
	"pggat2/lib/zap"
)

type ErrorResponse struct {
	Error perror.Error
}

func (T *ErrorResponse) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeErrorResponse {
		return false
	}

	var severity perror.Severity
	var code perror.Code
	var message string
	var extra []perror.ExtraField

	p := packet.Payload()

	for {
		var typ uint8
		p = p.ReadUint8(&typ)

		if typ == 0 {
			break
		}

		var value string
		p = p.ReadString(&value)

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

	T.Error = perror.New(
		severity,
		code,
		message,
		extra...,
	)
	return true
}

func (T *ErrorResponse) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeErrorResponse)

	packet = packet.AppendUint8('S')
	packet = packet.AppendString(string(T.Error.Severity()))

	packet = packet.AppendUint8('C')
	packet = packet.AppendString(string(T.Error.Code()))

	packet = packet.AppendUint8('M')
	packet = packet.AppendString(T.Error.Message())

	for _, field := range T.Error.Extra() {
		packet = packet.AppendUint8(uint8(field.Type))
		packet = packet.AppendString(field.Value)
	}

	packet = packet.AppendUint8(0)
	return packet
}
