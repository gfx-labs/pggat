package perror

import packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"

func ToPacket(err Error) *packets.ErrorResponse {
	var resp packets.ErrorResponse
	resp = append(
		resp,
		packets.ErrorResponseField{
			Code:  'S',
			Value: string(err.Severity()),
		},
		packets.ErrorResponseField{
			Code:  'C',
			Value: string(err.Code()),
		},
		packets.ErrorResponseField{
			Code:  'M',
			Value: err.Message(),
		},
	)
	extra := err.Extra()
	for _, field := range extra {
		resp = append(resp, packets.ErrorResponseField{
			Code:  uint8(field.Type),
			Value: field.Value,
		})
	}
	return &resp
}
