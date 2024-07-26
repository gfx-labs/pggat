package tracing

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"github.com/davecgh/go-spew/spew"
	"log/slog"
	"strings"
)

var packetTypeNames = map[fed.Type]string{
	packets.TypeQuery: "Query",
	packets.TypeClose: "Close",
	// packets.TypeCommandComplete: "CommandComplete",
	packets.TypeParameterStatus:    "ParameterStatus",
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
	packets.TypeErrorResponse:      "ErrorResponse",
	packets.TypeNoticeResponse:     "NoticeResponse",
}

func getPacketTypeName(t fed.Type) string {
	if str, ok := packetTypeNames[t]; ok {
		return str
	}

	return "<unknown type>"
}

type pgtrace struct {
	// span trace.Span
	logFunc map[int]func(msg string, packet fed.Packet)
}

func NewPgTrace(ctx context.Context) fed.Middleware {
	return &pgtrace{
		logFunc: map[int]func(msg string, packet fed.Packet){
			packets.TypeQuery:           logQuery,
			packets.TypeClose:           logClose,
			packets.TypeParameterStatus: logParameterStatus,
			packets.TypeRowDescription:  logRowDescription,
			packets.TypeDataRow:         logDataRow,
			packets.TypeReadyForQuery:   logReadyForQuery,
			packets.TypeErrorResponse:   logErrorResponse,
		},
	}
}

func (t *pgtrace) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *pgtrace) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	t.getLogFunc(packet)("ReadPacket ", packet)

	return packet, nil
}

func (t *pgtrace) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	t.getLogFunc(packet)("WritePacket", packet)

	return packet, nil
}

func (t *pgtrace) PostWrite(ctx context.Context) (fed.Packet, error) {
	return nil, nil
}

func (t *pgtrace) getLogFunc(packet fed.Packet) func(msg string, packet fed.Packet) {
	if lf, ok := t.logFunc[int(packet.Type())]; ok {
		return lf
	}

	return logDefault
}

func logQuery(msg string, packet fed.Packet) {
	sql := "<unknown>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		if q, err := fed.CloneDecoder(pp.Decoder, nil).String(); err == nil {
			sql = q
		}
	} else {
		if qp, ok := packet.(*packets.Query); ok && (qp != nil) {
			sql = string(*qp)
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
		var payload packets.Close

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
			"which", payload.Which,
			"name", payload.Name)
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
			spew.Dump(packet)
		}

		return
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
				spew.Dump(packet)
			}
		}

		return
	}
}

func logRowDescription(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var rowDescription packets.RowDescription

		if err := rowDescription.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err != nil {
			logDefault(msg, packet)
			return
		}

		typeName := getPacketTypeName(packet.Type())

		slog.Info(
			fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
			"type", packet.Type(),
			"typename", typeName,
			"length", packet.Length(),
			"rows", len(rowDescription))
	} else {
		slog.Warn(fmt.Sprintf("rowdesc: %T %#v", packet, packet))
		logDefault(msg, packet)
	}
}

func logDataRow(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var rowCount uint16
		var err error

		rowCount, err = fed.CloneDecoder(pp.Decoder, nil).Uint16()
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
			"rows", rowCount)
	} else {
		slog.Warn(fmt.Sprintf("rowdata: %T %#v", packet, packet))
		logDefault(msg, packet)
	}
}

func logReadyForQuery(msg string, packet fed.Packet) {
	if pp, ok := packet.(fed.PendingPacket); ok {
		var status uint8
		var err error

		status, err = fed.CloneDecoder(pp.Decoder, nil).Uint8()
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
		var errResponse packets.ErrorResponse

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

func logDefault(msg string, packet fed.Packet) {
	typeName := getPacketTypeName(packet.Type())

	slog.Info(
		fmt.Sprintf("%s: %s", msg, strings.ToUpper(typeName)),
		"type", packet.Type(),
		"typename", typeName,
		"length", packet.Length())
}
