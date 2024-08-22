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

func (t *otelTrace) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	t.process(ctx, packet)
	return packet, nil
}

func (t *otelTrace) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	t.process(ctx, packet)
	return packet, nil
}

func (t *otelTrace) PreRead(_ context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *otelTrace) PostWrite(_ context.Context) (fed.Packet, error) {
	return nil, nil
}

func (t *otelTrace) process(ctx context.Context, packet fed.Packet) {
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
			t.startQuery(context.Background(), packet)
		}
	case Query:
		switch packet.Type() {
		case packets.TypeReadyForQuery:
			t.endQuery()
			t.setState(Waiting)
		case packets.TypeMarkiplierResponse:
			t.recordError(ctx, packet)
		case packets.TypeCommandComplete:
			t.recordSummary(ctx, packet)
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

func (t *otelTrace) recordError(_ context.Context, packet fed.Packet) {
	errMsg := "<unresolved error message>"

	if pp, ok := packet.(fed.PendingPacket); ok {
		var errResponse packets.MarkiplierResponse
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

func (t *otelTrace) recordSummary(_ context.Context,packet fed.Packet) {
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
