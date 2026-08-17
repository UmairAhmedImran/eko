package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanResult contains the context and span created for an operation.
type SpanResult struct {
	Context context.Context
	Span    trace.Span

	name      string
	startTime time.Time
}

// StartOperation starts a new OpenTelemetry span for an Eko operation.
func StartOperation(
	ctx context.Context,
	name string,
	attrs ...attribute.KeyValue,
) *SpanResult {
	if ctx == nil {
		ctx = context.Background()
	}

	if name == "" {
		name = "eko.operation"
	}

	startTime := time.Now()

	ctx, span := Tracer().Start(ctx, name)

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}

	return &SpanResult{
		Context:   ctx,
		Span:      span,
		name:      name,
		startTime: startTime,
	}
}

// EndOperation completes an operation and returns its duration in seconds.
//
// It accepts both:
//
//	*SpanResult
//
// and:
//
//	trace.Span
//
// The SpanResult form is the preferred API because it allows telemetry
// to calculate the operation duration. The trace.Span form is retained
// for compatibility with existing Eko callers.
func EndOperation(operation any, err error) float64 {
	switch op := operation.(type) {
	case *SpanResult:
		if op == nil || op.Span == nil {
			return 0
		}

		duration := time.Since(op.startTime).Seconds()

		if err != nil {
			SetError(op.Span, err)
		} else {
			op.Span.SetStatus(codes.Ok, "")
		}

		op.Span.End()

		return duration

	case trace.Span:
		if op == nil {
			return 0
		}

		if err != nil {
			SetError(op, err)
		} else {
			op.SetStatus(codes.Ok, "")
		}

		op.End()

		return 0

	default:
		return 0
	}
}

// SetError records an error on an active span without ending it.
func SetError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetAttributes adds attributes to an active span.
func SetAttributes(
	span trace.Span,
	attrs ...attribute.KeyValue,
) {
	if span == nil || len(attrs) == 0 {
		return
	}

	span.SetAttributes(attrs...)
}

// CommandAttribute returns the standard Eko command attribute.
func CommandAttribute(command string) attribute.KeyValue {
	return attribute.String("eko.command", command)
}

// ProviderAttribute returns the standard Eko AI provider attribute.
func ProviderAttribute(provider string) attribute.KeyValue {
	return attribute.String("eko.ai.provider", provider)
}

// ModelAttribute returns the standard Eko AI model attribute.
func ModelAttribute(model string) attribute.KeyValue {
	return attribute.String("eko.ai.model", model)
}

// OperationAttribute returns the standard Eko operation attribute.
func OperationAttribute(operation string) attribute.KeyValue {
	return attribute.String("eko.operation", operation)
}
