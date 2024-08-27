package tracing

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"log/slog"
	"reflect"
)

// span trace.Span
var logFunc map[int]func(msg string, packet fed.Packet) = map[int]func(msg string, packet fed.Packet){
	packets.TypeQuery:              logQuery,
	packets.TypeClose:              logClose,
	packets.TypeParameterStatus:    logParameterStatus,
	packets.TypeRowDescription:     logRowDescription,
	packets.TypeDataRow:            logDataRow,
	packets.TypeReadyForQuery:      logReadyForQuery,
	packets.TypeMarkiplierResponse: logErrorResponse,
}

func logPacket(msg string, packet fed.Packet) {
	getLogFunc(packet)(msg, packet)
}

func getLogFunc(packet fed.Packet) func(msg string, packet fed.Packet) {
	if lf, ok := logFunc[int(packet.Type())]; ok {
		return lf
	}

	return logDefault
}

func logDefault(msg string, packet fed.Packet) {
	// 1 alloc
	slog.Info(
		msg,
		"type", packet.Type(),
		"typename", packet.TypeName(),
		"length", packet.Length())
}

func logQuery(msg string, packet fed.Packet) {
	sql := "<unresolved>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		if q, err := fed.CloneDecoder(pp.Decoder, nil).String(); err == nil {
			sql = q
		}
	} else {
		if qp, ok := packet.(*packets.Query); ok && (qp != nil) {
			// 0 alloc
			sql = string(*qp)
		} else {
			// 1 alloc
			slog.Warn(fmt.Sprintf("query: %s", reflect.TypeOf(packet)))
			logDefault(msg, packet)
			return
		}
	}

	// 2 alloc
	slog.Info(
		msg,
		"type", packet.Type(),
		"typename", packet.TypeName(),
		"length", packet.Length(),
		"sql", sql)
}

func logClose(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var payload packets.CommandComplete

		if err := payload.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err != nil {
			logDefault(msg, packet)
			return
		}

		// 2 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"typename", packet.TypeName(),
			"length", packet.Length(),
			"info", payload)
	} else {
		if cc, ok := packet.(*packets.CommandComplete); ok {
			info := string(*cc)

			// 2 alloc
			slog.Info(
				msg,
				"type", packet.Type(),
				"typename", packet.TypeName(),
				"length", packet.Length(),
				"info", info)

		} else {
			// 1 alloc
			slog.Warn(fmt.Sprintf("close: %s", reflect.TypeOf(packet)))
			logDefault(msg, packet)
		}
	}
}

func logParameterStatus(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var payload packets.ParameterStatus

		if err := payload.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err != nil {
			logDefault(msg, packet)
			return
		}

		// 3 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"length", packet.Length(),
			"key", payload.Key,
			"value", payload.Value)
	} else {
		if ps, ok := packet.(*packets.ParameterStatus); ok {
			// 3 alloc
			slog.Info(
				msg,
				"type", packet.Type(),
				"typename", packet.TypeName(),
				"length", packet.Length(),
				"key", ps.Key,
				"value", ps.Value)
		} else {
			if _, ok := packet.(*packets.Sync); ok {
				// 1 alloc
				slog.Info(
					msg,
					"type", packet.Type(),
					"typename", packet.TypeName(),
					"length", packet.Length())

			} else {
				// 1 alloc
				slog.Warn(fmt.Sprintf("paramstatus: %s", reflect.TypeOf(packet)))
				logDefault(msg, packet)
			}
		}
	}
}

func logRowDescription(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		columnCount, err := fed.CloneDecoder(pp.Decoder, nil).Uint16()
		if err != nil {
			logDefault(msg, packet)
			return
		}

		// 1 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"typename", packet.TypeName(),
			"length", packet.Length(),
			"#cols", columnCount)
	} else {
		// 1 alloc
		slog.Warn(fmt.Sprintf("rowdesc: %s", reflect.TypeOf(packet)))
		logDefault(msg, packet)
	}
}

func logDataRow(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		colCount, err := fed.CloneDecoder(pp.Decoder, nil).Uint16()
		if err != nil {
			logDefault(msg, packet)
			return
		}

		// 1 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"typename", packet.TypeName(),
			"length", packet.Length(),
			"#cols", colCount)
	} else {
		// 1 alloc
		slog.Warn(fmt.Sprintf("rowdata: %s", reflect.TypeOf(packet)))
		logDefault(msg, packet)
	}
}

func logReadyForQuery(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		status, err := fed.CloneDecoder(pp.Decoder, nil).Uint8()
		if err != nil {
			logDefault(msg, packet)
			return
		}

		// 2 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"typename", packet.TypeName(),
			"length", packet.Length(),
			"status", status)
	} else {
		if r4q, ok := packet.(*packets.ReadyForQuery); ok {
			// 2 alloc
			slog.Info(
				msg,
				"type", packet.Type(),
				"typename", packet.TypeName(),
				"length", packet.Length(),
				"status", *r4q)
		} else {
			// 1 alloc
			slog.Warn(fmt.Sprintf("rdy4qry: %s", reflect.TypeOf(packet)))
			logDefault(msg, packet)
		}
	}
}

func logErrorResponse(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var errResponse packets.MarkiplierResponse

		if err := errResponse.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err != nil {
			logDefault(msg, packet)
			return
		}

		var errMsg string
		for _, resp := range errResponse {
			if resp.Code == 77 {
				errMsg = resp.Value
				break
			}
		}

		// 2 alloc
		slog.Info(
			msg,
			"type", packet.Type(),
			"typename", packet.TypeName(),
			"length", packet.Length(),
			"count", len(errResponse),
			"error", errMsg)
	} else {
		// 1 alloc
		slog.Warn(fmt.Sprintf("errrspnse: %s", reflect.TypeOf(packet)))
		logDefault(msg, packet)
	}
}
