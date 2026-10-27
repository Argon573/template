package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName string
	Endpoint    string
	Protocol    string
	Sampler     string
	LogLevel    string
}

type Provider struct {
	Logger         *slog.Logger
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	TripMetrics    *TripMetrics
	HTTPMetrics    *HTTPMetrics
}

type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func (m *TripMetrics) IncCreated(ctx context.Context) {
	m.created.Add(ctx, 1)
}

func (m *TripMetrics) IncCompleted(ctx context.Context) {
	m.completed.Add(ctx, 1)
}

type HTTPMetrics struct {
	Duration metric.Float64Histogram
	Active   metric.Int64UpDownCounter
}

func (m *HTTPMetrics) IncActive(ctx context.Context, method, route string) {
	m.Active.Add(ctx, 1, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
}

func (m *HTTPMetrics) DecActive(ctx context.Context, method, route string) {
	m.Active.Add(ctx, -1, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
}

func (m *HTTPMetrics) RecordDuration(ctx context.Context, seconds float64, method, route string, status int) {
	m.Duration.Record(ctx, seconds, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.String("http.response.status_code", strconv.Itoa(status)),
	))
}

func New(ctx context.Context, cfg Config) (*Provider, error) {
	logger := newLogger(cfg.ServiceName, cfg.LogLevel)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceNamespace("tripgo"),
			semconv.DeploymentEnvironment("local"),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	endpoint := strings.TrimPrefix(cfg.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(cfg.ServiceName)

	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create trips.created metric: %w", err)
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create trips.completed metric: %w", err)
	}

	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
	)
	if err != nil {
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create request.duration metric: %w", err)
	}

	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create active_requests metric: %w", err)
	}

	return &Provider{
		Logger:         logger,
		TracerProvider: tp,
		MeterProvider:  mp,
		TripMetrics: &TripMetrics{
			created:   created,
			completed: completed,
		},
		HTTPMetrics: &HTTPMetrics{
			Duration: duration,
			Active:   active,
		},
	}, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	var firstErr error
	if err := p.MeterProvider.Shutdown(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := p.TracerProvider.Shutdown(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func newLogger(serviceName, level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	base := slog.New(handler).With("service", serviceName)
	return slog.New(&traceHandler{Handler: base.Handler()})
}

type traceHandler struct {
	slog.Handler
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	} else {
		record.AddAttrs(
			slog.String("trace_id", ""),
			slog.String("span_id", ""),
		)
	}
	return h.Handler.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name)}
}
