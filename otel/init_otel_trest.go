package otel

import (
	"context"
	"testing"
	"time"
)

func TestInitOtelProvider(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: true,
			Type:   ExporterTypeStdout,
			Pretty: false,
		},
		Trace: TraceConfig{
			Enable:        true,
			Type:          ExporterTypeStdout,
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:               true,
			Type:                 ExporterTypeStdout,
			Pretty:               false,
			IntervalSeconds:      5,
			EnableRuntimeMetrics: false,
		},
	}

	err := InitOtelProvider(config)
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if globalProvider == nil {
		t.Fatal("global provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	// Reset global provider for other tests
	globalProvider = nil
}

func TestGetOtelProvider(t *testing.T) {
	// Initially should be nil
	globalProvider = nil
	if GetOtelProvider() != nil {
		t.Error("global provider should be nil initially")
	}

	config := DefaultConfig()
	config.ServiceName = "test-service"

	if err := InitOtelProvider(config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	provider := GetOtelProvider()
	if provider == nil {
		t.Fatal("global provider is nil after initialization")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

func TestShutdownOtelProvider_NoProvider(t *testing.T) {
	globalProvider = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not error when no provider is set
	if err := ShutdownOtelProvider(ctx); err != nil {
		t.Errorf("shutdown should not error when no provider is set: %v", err)
	}
}

func TestGetProviders(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"

	if err := InitOtelProvider(config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	provider := GetOtelProvider()
	if provider == nil {
		t.Fatal("global provider is nil")
	}

	if provider.GetLoggerProvider() == nil {
		t.Error("logger provider is nil")
	}

	if provider.GetTracerProvider() == nil {
		t.Error("tracer provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Error("meter provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

func TestInitOtelProvider_WithDifferentConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "Only Log Enabled",
			config: &Config{
				ServiceName:    "test-log-only",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable: true,
					Type:   ExporterTypeStdout,
				},
				Trace: TraceConfig{
					Enable: false,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			},
		},
		{
			name: "Only Trace Enabled",
			config: &Config{
				ServiceName:    "test-trace-only",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable: false,
				},
				Trace: TraceConfig{
					Enable:        true,
					Type:          ExporterTypeStdout,
					SamplingRatio: 0.5,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			},
		},
		{
			name: "Only Metric Enabled",
			config: &Config{
				ServiceName:    "test-metric-only",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable: false,
				},
				Trace: TraceConfig{
					Enable: false,
				},
				Metric: MetricConfig{
					Enable:               true,
					Type:                 ExporterTypeStdout,
					IntervalSeconds:      5,
					EnableRuntimeMetrics: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalProvider = nil

			err := InitOtelProvider(tt.config)
			if err != nil {
				t.Fatalf("failed to initialize: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := ShutdownOtelProvider(ctx); err != nil {
				t.Errorf("failed to shutdown: %v", err)
			}

			globalProvider = nil
		})
	}
}

