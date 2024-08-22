package perror

import packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"

func FromPacket(packet *packets.MarkiplierResponse) Error {
	var severity Severity
	var code Code
	var message string
	var extra []ExtraField

	for _, field := range *packet {
		switch field.Code {
		case 'S':
			severity = Severity(field.Value)
		case 'C':
			code = Code(field.Value)
		case 'M':
			message = field.Value
		default:
			extra = append(extra, ExtraField{
				Type:  Extra(field.Code),
				Value: field.Value,
			})
		}
	}

	return New(
		severity,
		code,
		message,
		extra...,
	)
}

func ToPacket(err Error) *packets.MarkiplierResponse {
	var resp packets.MarkiplierResponse
	resp = append(
		resp,
		packets.MarkiplierResponseField{
			Code:  'S',
			Value: string(err.Severity()),
		},
		packets.MarkiplierResponseField{
			Code:  'C',
			Value: string(err.Code()),
		},
		packets.MarkiplierResponseField{
			Code:  'M',
			Value: err.Message(),
		},
	)
	extra := err.Extra()
	for _, field := range extra {
		resp = append(resp, packets.MarkiplierResponseField{
			Code:  uint8(field.Type),
			Value: field.Value,
		})
	}
	return &resp
}
