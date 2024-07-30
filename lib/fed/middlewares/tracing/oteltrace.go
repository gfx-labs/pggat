package tracing

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type queryState int

const (
	Init queryState = iota
	Waiting
	Query
)

type otelTrace struct {
	state  queryState
	tracer trace.Tracer
	span   trace.Span
}

func NewOtelTrace() fed.Middleware {
	return &otelTrace{
		tracer: otel.Tracer(
			"pggat",
			trace.WithInstrumentationAttributes(
				attribute.String("component", "github.com/gfx.labs/pggat"))),
	}
}

func (t *otelTrace) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	t.process(packet)
	return packet, nil
}

func (t *otelTrace) WritePacket(packet fed.Packet) (fed.Packet, error) {
	t.process(packet)
	return packet, nil
}

func (t *otelTrace) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *otelTrace) PostWrite() (fed.Packet, error) {
	return nil, nil
}

func (t *otelTrace) process(packet fed.Packet) {
	switch t.state {
	case Init:
		switch packet.Type() {
		case packets.TypeReadyForQuery:
			t.setState(Waiting)
		}
	case Waiting:
		switch packet.Type() {
		case packets.TypeQuery:
			t.setState(Query)
			t.startQuery(context.Background(),packet)
		}
	case Query:
		switch packet.Type() {
		case packets.TypeReadyForQuery:
			t.endQuery()
			t.setState(Waiting)
		case packets.TypeErrorResponse:
			t.recordError(packet)
		case packets.TypeCommandComplete:
			t.recordSummary(packet)
		}
	}
}

func getStateName(state queryState) (str string) {
	switch state {
	case Init:
		str = "Init"
	case Waiting:
		str = "Waiting"
	case Query:
		str = "Query"
	default:
		str = "<unknown>"
	}

	return
}

func (t *otelTrace) setState(state queryState) {
	// slog.Warn(fmt.Sprintf("State Change: %s => %s", getStateName(t.state), getStateName(state)))
	t.state = state
}

func (t *otelTrace) startQuery(ctx context.Context, packet fed.Packet) {
	sql := "<unresolved>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		if q, err := fed.CloneDecoder(pp.Decoder, nil).String(); err == nil {
			sql = q
		}
	} else {
		if qp, ok := packet.(*packets.Query); ok && (qp != nil) {
			sql = string(*qp)
		}
	}

	t.endQuery()

	_, t.span = t.tracer.Start(ctx, "Query",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String("sql", sql)))
}

func (t *otelTrace) endQuery() {
	if t.span != nil {
		t.span.End()
		t.span = nil
	}
}

func (t *otelTrace) recordError(packet fed.Packet) {
	errMsg := "<unresolved error message>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		var errResponse packets.ErrorResponse
		if err := errResponse.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err == nil {
			for _, resp := range errResponse {
				if resp.Code == 77 {
					errMsg = resp.Value
					break
				}
			}
		}
	}

	t.span.RecordError(fmt.Errorf(errMsg))
	t.span.SetStatus(codes.Error, errMsg)
}

func (t *otelTrace) recordSummary(packet fed.Packet) {
	summary := "<unresolved query summary>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		var payload packets.CommandComplete

		if err := payload.ReadFrom(fed.CloneDecoder(pp.Decoder, nil)); err == nil {
			summary = string(payload)
		}
	} else {
		if cc, ok := packet.(*packets.CommandComplete); ok {
			summary = string(*cc)
		}
	}

	t.span.SetAttributes(attribute.String("summary", summary))
}
