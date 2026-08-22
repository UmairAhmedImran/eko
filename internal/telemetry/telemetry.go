package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName    = "eko"
	instrumentName = "eko/telemetry"

	defaultOTLPEndpoint = "http://localhost:4318"

	telemetryTimeout        = 5 * time.Second
	metricExportInterval    = 5 * time.Second
	traceExportBatchTimeout = 5 * time.Second
)

var (
	initOnce sync.Once

	initErr error

	shutdownMu sync.Mutex
	shutdown   func(context.Context) error

	tracer trace.Tracer
	meter  metric.Meter
)

// Init initializes OpenTelemetry for Eko.
//
// Telemetry is disabled unless:
//
//	EKO_OTEL_ENABLED=true
//
// When enabled, Eko exports metrics and traces through an OpenTelemetry
// Collector using OTLP over HTTP.
//
// The base endpoint is configured with:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT
//
// Example:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
//
// The following signal endpoints are generated automatically:
//
//	http://localhost:4318/v1/metrics
//	http://localhost:4318/v1/traces
//
// Eko is a short-lived CLI, so the returned shutdown function explicitly
// flushes pending metrics and traces before the process exits.
func Init(ctx context.Context) (func(context.Context) error, error) {
	initOnce.Do(func() {
		// Always initialize safe no-op API handles first.
		tracer = otel.Tracer(instrumentName)
		meter = otel.Meter(instrumentName)

		// Telemetry is opt-in.
		if !telemetryEnabled() {
			if err := initMetrics(); err != nil {
				initErr = fmt.Errorf(
					"initialize disabled telemetry metrics: %w",
					err,
				)
				return
			}

			shutdown = func(context.Context) error {
				return nil
			}

			return
		}

		if ctx == nil {
			ctx = context.Background()
		}

		baseEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if strings.TrimSpace(baseEndpoint) == "" {
			baseEndpoint = defaultOTLPEndpoint
		}

		metricsEndpoint, tracesEndpoint, err := buildEndpoints(baseEndpoint)
		if err != nil {
			initErr = fmt.Errorf(
				"invalid OTLP endpoint: %w",
				err,
			)
			return
		}

		// ------------------------------------------------------------
		// Resource
		// ------------------------------------------------------------

		res, err := newResource(ctx)
		if err != nil {
			initErr = fmt.Errorf(
				"create telemetry resource: %w",
				err,
			)
			return
		}

		// ------------------------------------------------------------
		// Trace exporter
		// ------------------------------------------------------------

		traceExporterOptions := []otlptracehttp.Option{
			otlptracehttp.WithEndpointURL(tracesEndpoint),
			otlptracehttp.WithTimeout(telemetryTimeout),
		}

		if isInsecureEndpoint(tracesEndpoint) {
			traceExporterOptions = append(
				traceExporterOptions,
				otlptracehttp.WithInsecure(),
			)
		}

		traceExporter, err := otlptracehttp.New(
			ctx,
			traceExporterOptions...,
		)
		if err != nil {
			initErr = fmt.Errorf(
				"create OTLP trace exporter: %w",
				err,
			)
			return
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(
				traceExporter,
				sdktrace.WithBatchTimeout(
					traceExportBatchTimeout,
				),
			),
			sdktrace.WithResource(res),
		)

		// ------------------------------------------------------------
		// Metrics exporter
		// ------------------------------------------------------------

		metricExporterOptions := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpointURL(metricsEndpoint),
			otlpmetrichttp.WithTimeout(telemetryTimeout),
		}

		if isInsecureEndpoint(metricsEndpoint) {
			metricExporterOptions = append(
				metricExporterOptions,
				otlpmetrichttp.WithInsecure(),
			)
		}

		metricExporter, err := otlpmetrichttp.New(
			ctx,
			metricExporterOptions...,
		)
		if err != nil {
			_ = tp.Shutdown(context.Background())

			initErr = fmt.Errorf(
				"create OTLP metric exporter: %w",
				err,
			)
			return
		}

		reader := sdkmetric.NewPeriodicReader(
			metricExporter,
			sdkmetric.WithInterval(metricExportInterval),
			sdkmetric.WithTimeout(telemetryTimeout),
		)

		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(res),
		)

		// Register providers globally before creating instruments.
		otel.SetTracerProvider(tp)
		otel.SetMeterProvider(mp)

		tracer = tp.Tracer(instrumentName)
		meter = mp.Meter(instrumentName)

		// Application metric instruments must be created from the
		// configured MeterProvider.
		if err := initMetrics(); err != nil {
			_ = mp.Shutdown(context.Background())
			_ = tp.Shutdown(context.Background())

			initErr = fmt.Errorf(
				"initialize application metrics: %w",
				err,
			)
			return
		}

		// ------------------------------------------------------------
		// Shutdown
		// ------------------------------------------------------------

		var shutdownOnce sync.Once
		var shutdownErr error

		shutdown = func(shutdownCtx context.Context) error {
			shutdownOnce.Do(func() {
				shutdownErr = shutdownTelemetry(
					shutdownCtx,
					reader,
					tp,
					mp,
				)
			})

			return shutdownErr
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	if shutdown == nil {
		return nil, fmt.Errorf(
			"telemetry initialization completed without shutdown handler",
		)
	}

	// Return a wrapper so the global shutdown function remains protected
	// from accidental reassignment.
	shutdownMu.Lock()
	currentShutdown := shutdown
	shutdownMu.Unlock()

	return currentShutdown, nil
}

// telemetryEnabled determines whether Eko telemetry is enabled.
//
// Telemetry is intentionally opt-in because Eko is a CLI and should not
// unexpectedly send telemetry anywhere.
func telemetryEnabled() bool {
	return strings.EqualFold(
		strings.TrimSpace(os.Getenv("EKO_OTEL_ENABLED")),
		"true",
	)
}

// newResource creates the resource associated with all Eko telemetry.
//
// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES are respected through
// resource.WithFromEnv(). The code-level Eko service name provides a
// deterministic fallback.
func newResource(ctx context.Context) (*resource.Resource, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// shutdownTelemetry flushes and shuts down all telemetry providers.
//
// Metrics and traces are flushed explicitly because a CLI process can
// terminate before periodic/batched exporters have exported their data.
func shutdownTelemetry(
	ctx context.Context,
	reader *sdkmetric.PeriodicReader,
	tp *sdktrace.TracerProvider,
	mp *sdkmetric.MeterProvider,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var firstErr error

	if err := reader.ForceFlush(ctx); err != nil {
		firstErr = err
	}

	if err := tp.ForceFlush(ctx); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := mp.Shutdown(ctx); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := tp.Shutdown(ctx); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// buildEndpoints converts a base OTLP endpoint into explicit signal
// endpoints.
//
// Input:
//
//	http://localhost:4318
//
// Output:
//
//	http://localhost:4318/v1/metrics
//	http://localhost:4318/v1/traces
func buildEndpoints(
	baseEndpoint string,
) (metricsEndpoint string, tracesEndpoint string, err error) {
	baseEndpoint = strings.TrimSpace(baseEndpoint)

	if baseEndpoint == "" {
		return "", "", fmt.Errorf("endpoint is empty")
	}

	parsed, err := url.Parse(baseEndpoint)
	if err != nil {
		return "", "", fmt.Errorf("parse endpoint: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf(
			"endpoint scheme must be http or https, got %q",
			parsed.Scheme,
		)
	}

	if parsed.Host == "" {
		return "", "", fmt.Errorf(
			"endpoint must include a host",
		)
	}

	path := strings.TrimRight(parsed.Path, "/")

	switch path {
	case "":
		// Base OTLP endpoint.
	case "/v1/metrics":
		path = ""
	case "/v1/traces":
		path = ""
	default:
		return "", "", fmt.Errorf(
			"unsupported endpoint path %q; expected an OTLP base endpoint",
			parsed.Path,
		)
	}

	parsed.Path = path

	metricsURL := *parsed
	metricsURL.Path = "/v1/metrics"

	tracesURL := *parsed
	tracesURL.Path = "/v1/traces"

	return metricsURL.String(), tracesURL.String(), nil
}

// isInsecureEndpoint determines whether TLS should be disabled.
//
// OTLP HTTP exporters require WithInsecure() for plain HTTP endpoints.
// HTTPS endpoints use TLS normally.
func isInsecureEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	return strings.EqualFold(parsed.Scheme, "http")
}

// Tracer returns Eko's shared OpenTelemetry tracer.
func Tracer() trace.Tracer {
	if tracer == nil {
		tracer = otel.Tracer(instrumentName)
	}

	return tracer
}

// Meter returns Eko's shared OpenTelemetry meter.
func Meter() metric.Meter {
	if meter == nil {
		meter = otel.Meter(instrumentName)
	}

	return meter
}

// StartSpan starts a span using Eko's shared tracer.
func StartSpan(
	ctx context.Context,
	name string,
) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	return Tracer().Start(ctx, name)
}
