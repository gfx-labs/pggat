package backends

import (
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func Query(server zap.ReadWriter, query string) error {
	packet := zap.NewPacket()
	packet.WriteType(packets.Query)
	packet.WriteString(query)
	err := server.Write(packet)
	if err != nil {
		return err
	}

	for {
		err = server.Read(packet)
		if err != nil {
			return err
		}

		switch packet.ReadType() {
		case packets.CommandComplete,
			packets.RowDescription,
			packets.DataRow,
			packets.EmptyQueryResponse,
			packets.ErrorResponse,
			packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			continue
		case packets.CopyInResponse:
			// send copy fail
			packet.WriteType(packets.CopyFail)
			packet.WriteString("unexpected")
			if err = server.Write(packet); err != nil {
				return err
			}
		case packets.CopyOutResponse:
		outer:
			for {
				err = server.Read(packet)
				if err != nil {
					return err
				}

				switch packet.ReadType() {
				case packets.CopyData,
					packets.NoticeResponse,
					packets.ParameterStatus,
					packets.NotificationResponse:
					continue
				case packets.CopyDone, packets.ErrorResponse:
					break outer
				default:
					return ErrUnexpectedPacket
				}
			}
		case packets.ReadyForQuery:
			read := packet.Read()
			state, ok := packets.ReadReadyForQuery(&read)
			if !ok || state != 'I' {
				return ErrUnexpectedPacket
			}
			return nil
		default:
			return ErrUnexpectedPacket
		}
	}
}
