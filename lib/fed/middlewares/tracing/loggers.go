package tracing

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"log/slog"
	"strings"
)

var packetTypeNames = map[fed.Type]string{
	packets.TypeQuery: "Query",
	// packets.TypeClose: "Close",
	packets.TypeCommandComplete: "CommandComplete",
	packets.TypeParameterStatus: "ParameterStatus",
	// packets.TypeSync: "Sync",
	packets.TypeReadyForQuery:      "ReadyForQuery",
	packets.TypeRowDescription:     "RowDescription",
	packets.TypeDataRow:            "DataRow",
	packets.TypeEmptyQueryResponse: "EmptyQueryResponse",
	packets.TypeAuthentication:     "Authentication",
	packets.TypeBind:               "Bind",
	packets.TypeBindComplete:       "BindComplete",
	packets.TypeCloseComplete:      "CloseComplete",
	packets.TypeCopyData:           "CopyData",
	packets.TypeCopyDone:           "CopyDone",
	packets.TypeCopyFail:           "CopyFail",
	packets.TypeMarkiplierResponse: "ErrorResponse",
	// packets.TypeExecute: "Execute",
	packets.TypeNoticeResponse:       "NoticeResponse",
	packets.TypeFlush:                "Flush",
	packets.TypeTerminate:            "Terminate",
	packets.TypeParse:                "Parse",
	packets.TypeParseComplete:        "ParseComplete",
	packets.TypeFunctionCall:         "FunctionCall",
	packets.TypeFunctionCallResponse: "FunctionCallResponse",
}

func getPacketTypeName(t fed.Type) string {
	if str, ok := packetTypeNames[t]; ok {
		return str
	}

	return "<unknown type>"
}

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
	typeName := getPacketTypeName(packet.Type())

	slog.Info(
		fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
		"type", packet.Type(),
		"typename", typeName,
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
			sql = string(*qp)
		} else {
			slog.Warn(fmt.Sprintf("query: %T %#v", packet, packet))
			logDefault(msg, packet)
			return
		}
	}

	typeName := getPacketTypeName(packet.Type())

	slog.Info(
		fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
		"type", packet.Type(),
		"typename", typeName,
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

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"info", payload)
	} else {
		if cc, ok := packet.(*packets.CommandComplete); ok {
			typeName := "CommandComplete"

			info := string(*cc)

			slog.Info(
				fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
				"type", packet.Type(),
				"typename", typeName,
				"length", packet.Length(),
				"info", info)

		} else {
			slog.Warn(fmt.Sprintf("close: %T %#v", packet, packet))
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

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"key", payload.Key,
			"value", payload.Value)
	} else {
		if ps, ok := packet.(*packets.ParameterStatus); ok {
			typeName := getPacketTypeName(packet.Type())

			slog.Info(
				fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
				"type", packet.Type(),
				"typename", typeName,
				"length", packet.Length(),
				"key", ps.Key,
				"value", ps.Value)
		} else {
			if _, ok := packet.(*packets.Sync); ok {
				typeName := "Sync"

				slog.Info(
					fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
					"type", packet.Type(),
					"typename", typeName,
					"length", packet.Length())

			} else {
				slog.Warn(fmt.Sprintf("paramstatus: %T %#v", packet, packet))
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

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"#cols", columnCount)
	} else {
		slog.Warn(fmt.Sprintf("rowdesc: %T %#v", packet, packet))
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

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"#cols", colCount)
	} else {
		slog.Warn(fmt.Sprintf("rowdata: %T %#v", packet, packet))
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

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"status", status)
	} else {
		if r4q, ok := packet.(*packets.ReadyForQuery); ok {
			typeName := getPacketTypeName(packet.Type())

			slog.Info(
				fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
				"type", packet.Type(),
				"typename", typeName,
				"length", packet.Length(),
				"status", *r4q)
		} else {
			slog.Warn(fmt.Sprintf("rdy4qry: %T %#v", packet, packet))
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

		typeName := getPacketTypeName(packet.Type())

		var errMsg string
		for _, resp := range errResponse {
			if resp.Code == 77 {
				errMsg = resp.Value
				break
			}
		}

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"count", len(errResponse),
			"error", errMsg)
	} else {
		slog.Warn(fmt.Sprintf("errrspnse: %T %#v", packet, packet))
		logDefault(msg, packet)
	}
}
