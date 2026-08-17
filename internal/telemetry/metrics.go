package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const metricScope = instrumentName

// Duration buckets are expressed in seconds.
var durationBuckets = []float64{
	0.0005, // 0.5 ms
	0.001,  // 1 ms
	0.0025, // 2.5 ms
	0.005,  // 5 ms
	0.01,   // 10 ms
	0.025,  // 25 ms
	0.05,   // 50 ms
	0.1,    // 100 ms
	0.25,   // 250 ms
	0.5,    // 500 ms
	1.0,    // 1 s
	2.5,    // 2.5 s
	5.0,    // 5 s
	10.0,   // 10 s
}

type Metrics struct {
	CommandTotal    otelmetric.Int64Counter
	CommandDuration otelmetric.Float64Histogram

	LLMTotal    otelmetric.Int64Counter
	LLMDuration otelmetric.Float64Histogram

	CASOperations otelmetric.Int64Counter
	CASDuration   otelmetric.Float64Histogram

	SQLiteOperations otelmetric.Int64Counter
	SQLiteDuration   otelmetric.Float64Histogram
}

var (
	metrics     Metrics
	metricsOnce sync.Once
	metricsErr  error
)

// initMetrics initializes all Eko application metrics.
//
// Metrics are initialized once and are safe to call repeatedly.
// This function is intentionally independent from whether telemetry
// exporters are enabled. Tests and application code can therefore
// safely record metrics without causing nil-instrument panics.
func initMetrics() error {
	metricsOnce.Do(func() {
		meter := Meter()

		metrics.CommandTotal, metricsErr = meter.Int64Counter(
			"eko_command_total",
			otelmetric.WithDescription(
				"Total number of Eko CLI commands executed.",
			),
		)
		if metricsErr != nil {
			return
		}

		metrics.CommandDuration, metricsErr = meter.Float64Histogram(
			"eko_command_duration_seconds",
			otelmetric.WithDescription(
				"Duration of Eko CLI commands in seconds.",
			),
			otelmetric.WithExplicitBucketBoundaries(durationBuckets...),
		)
		if metricsErr != nil {
			return
		}

		metrics.LLMTotal, metricsErr = meter.Int64Counter(
			"eko_llm_operations_total",
			otelmetric.WithDescription(
				"Total number of LLM operations.",
			),
		)
		if metricsErr != nil {
			return
		}

		metrics.LLMDuration, metricsErr = meter.Float64Histogram(
			"eko_llm_duration_seconds",
			otelmetric.WithDescription(
				"Duration of LLM operations in seconds.",
			),
			otelmetric.WithExplicitBucketBoundaries(durationBuckets...),
		)
		if metricsErr != nil {
			return
		}

		metrics.CASOperations, metricsErr = meter.Int64Counter(
			"eko_cas_operations_total",
			otelmetric.WithDescription(
				"Total number of content-addressable storage operations.",
			),
		)
		if metricsErr != nil {
			return
		}

		metrics.CASDuration, metricsErr = meter.Float64Histogram(
			"eko_cas_duration_seconds",
			otelmetric.WithDescription(
				"Duration of content-addressable storage operations in seconds.",
			),
			otelmetric.WithExplicitBucketBoundaries(durationBuckets...),
		)
		if metricsErr != nil {
			return
		}

		metrics.SQLiteOperations, metricsErr = meter.Int64Counter(
			"eko_sqlite_operations_total",
			otelmetric.WithDescription(
				"Total number of SQLite operations.",
			),
		)
		if metricsErr != nil {
			return
		}

		metrics.SQLiteDuration, metricsErr = meter.Float64Histogram(
			"eko_sqlite_duration_seconds",
			otelmetric.WithDescription(
				"Duration of SQLite operations in seconds.",
			),
			otelmetric.WithExplicitBucketBoundaries(durationBuckets...),
		)
	})

	if metricsErr != nil {
		return fmt.Errorf("initialize Eko metrics: %w", metricsErr)
	}

	return nil
}

// MetricsInstance returns the initialized metrics.
func MetricsInstance() Metrics {
	_ = initMetrics()
	return metrics
}

// durationSeconds converts the duration representations used by
// existing Eko callers into seconds.
//
// Existing code historically passed time.Time values representing
// operation start times. Newer code can pass a float64 duration.
//
// Supporting both here allows telemetry to evolve without requiring
// unrelated packages to change their APIs.
func durationSeconds(value any) float64 {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0
		}
		return v

	case float32:
		if v < 0 {
			return 0
		}
		return float64(v)

	case time.Duration:
		if v < 0 {
			return 0
		}
		return v.Seconds()

	case time.Time:
		d := time.Since(v)
		if d < 0 {
			return 0
		}
		return d.Seconds()

	default:
		return 0
	}
}

// RecordCommand records a completed CLI command.
//
// The duration argument accepts both:
//   - float64 seconds
//   - time.Time operation start time
func RecordCommand(
	ctx context.Context,
	command string,
	duration any,
	success bool,
) {
	_ = initMetrics()

	if ctx == nil {
		ctx = context.Background()
	}

	seconds := durationSeconds(duration)

	attrs := otelmetric.WithAttributes(
		attribute.String("command", command),
		attribute.Bool("success", success),
	)

	metrics.CommandTotal.Add(ctx, 1, attrs)
	metrics.CommandDuration.Record(ctx, seconds, attrs)
}

// RecordLLM records a completed LLM operation.
//
// Supported forms:
//
//	RecordLLM(ctx, operation, duration, success)
//
// and the legacy form:
//
//	RecordLLM(ctx, operation, model, start, success)
func RecordLLM(
	ctx context.Context,
	operation string,
	args ...any,
) {
	_ = initMetrics()

	if ctx == nil {
		ctx = context.Background()
	}

	var (
		model   string
		start   any
		success bool
	)

	switch len(args) {
	case 2:
		start = args[0]
		success, _ = args[1].(bool)

	case 3:
		model, _ = args[0].(string)
		start = args[1]
		success, _ = args[2].(bool)

	default:
		return
	}

	seconds := durationSeconds(start)

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	}

	if model != "" {
		attrs = append(attrs, attribute.String("model", model))
	}

	metricAttrs := otelmetric.WithAttributes(attrs...)

	metrics.LLMTotal.Add(ctx, 1, metricAttrs)
	metrics.LLMDuration.Record(ctx, seconds, metricAttrs)
}

// RecordCAS records a completed CAS operation.
//
// The duration argument accepts both:
//   - float64 seconds
//   - time.Time operation start time
func RecordCAS(
	ctx context.Context,
	operation string,
	duration any,
	success bool,
) {
	_ = initMetrics()

	if ctx == nil {
		ctx = context.Background()
	}

	seconds := durationSeconds(duration)

	attrs := otelmetric.WithAttributes(
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	)

	metrics.CASOperations.Add(ctx, 1, attrs)
	metrics.CASDuration.Record(ctx, seconds, attrs)
}

// RecordSQLite records a completed SQLite operation.
//
// The duration argument accepts both:
//   - float64 seconds
//   - time.Time operation start time
func RecordSQLite(
	ctx context.Context,
	operation string,
	duration any,
	success bool,
) {
	_ = initMetrics()

	if ctx == nil {
		ctx = context.Background()
	}

	seconds := durationSeconds(duration)

	attrs := otelmetric.WithAttributes(
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	)

	metrics.SQLiteOperations.Add(ctx, 1, attrs)
	metrics.SQLiteDuration.Record(ctx, seconds, attrs)
}
