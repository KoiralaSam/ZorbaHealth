package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Config configures the OpenTelemetry logging pipeline.
type Config struct {
	ServiceName  string
	Environment  string
	OTLPEndpoint string
}

var (
	initMu   sync.Mutex
	provider *sdklog.LoggerProvider
)

// InitLogger initializes an OTel LoggerProvider and returns a slog.Logger
// that exports to OTLP (when an endpoint is configured) and always mirrors
// to stdout. The returned shutdown function flushes pending records.
func InitLogger(cfg Config) (*slog.Logger, func(context.Context) error, error) {
	initMu.Lock()
	defer initMu.Unlock()

	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Only export OTLP logs when an endpoint is explicitly configured.
	// Do not derive from JAEGER_ENDPOINT — Jaeger accepts traces, not logs.
	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"))
	}

	if endpoint == "" {
		logger := slog.New(stdoutHandler).With(
			"service", cfg.ServiceName,
			"environment", cfg.Environment,
		)
		return logger, func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create log resource: %w", err)
	}

	exporter, err := newLogExporter(context.Background(), endpoint)
	if err != nil {
		return nil, nil, err
	}

	provider = sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	otelHandler := otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(provider))
	logger := slog.New(newFanoutHandler(stdoutHandler, otelHandler)).With(
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
	)

	shutdown := func(ctx context.Context) error {
		initMu.Lock()
		defer initMu.Unlock()
		if provider == nil {
			return nil
		}
		err := provider.Shutdown(ctx)
		provider = nil
		return err
	}
	return logger, shutdown, nil
}

func newLogExporter(ctx context.Context, endpoint string) (*otlploghttp.Exporter, error) {
	options := []otlploghttp.Option{}
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Scheme != "" {
		options = append(options, otlploghttp.WithEndpointURL(endpoint))
		if parsed.Scheme == "http" {
			options = append(options, otlploghttp.WithInsecure())
		}
	} else {
		options = append(options, otlploghttp.WithEndpoint(endpoint), otlploghttp.WithInsecure())
	}
	exporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP log exporter: %w", err)
	}
	return exporter, nil
}

// fanoutHandler writes every record to multiple slog handlers.
type fanoutHandler struct {
	handlers []slog.Handler
}

func newFanoutHandler(handlers ...slog.Handler) *fanoutHandler {
	return &fanoutHandler{handlers: handlers}
}

func (f *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var firstErr error
	for _, h := range f.handlers {
		if !h.Enabled(ctx, record.Level) {
			continue
		}
		if err := h.Handle(ctx, record.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: next}
}

func (f *fanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		next[i] = h.WithGroup(name)
	}
	return &fanoutHandler{handlers: next}
}
