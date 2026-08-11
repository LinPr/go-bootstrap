package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var globalProvider *OtelProviders

// NewOtelProviders initializes the OpenTelemetry SDK with the provided config.
// It is a convenience helper that sets and returns the global providers.
func NewOtelProviders(config *Config) (*OtelProviders, error) {
	providers, err := newOtelProviders(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	globalProvider = providers
	return providers, nil
}

// Shutdown closes the global OpenTelemetry providers.
func ShutdownOtelProvider(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}

// GetOtelProvider returns the global providers.
func GetOtelProvider() *OtelProviders {
	return globalProvider
}

// GetLoggerProvider returns the log provider.
func (p *OtelProviders) GetLoggerProvider() log.LoggerProvider {
	return p.logProvider
}

// GetTracerProvider returns the trace provider.
func (p *OtelProviders) GetTracerProvider() trace.TracerProvider {
	return p.traceProvider
}

// GetMeterProvider returns the metric provider.
func (p *OtelProviders) GetMeterProvider() metric.MeterProvider {
	return p.metricProvider
}
