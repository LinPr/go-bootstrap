package gobootstrap

import (
	"context"
	"testing"
	"time"
)

func TestNewProvider_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.ServiceVersion = "1.0.0"

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if provider == nil {
		t.Fatal("provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestNewProvider_StdoutExporters(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: true,
			Type:   ExporterTypeStdout,
			Pretty: true,
		},
		Trace: TraceConfig{
			Enable:        true,
			Type:          ExporterTypeStdout,
			Pretty:        true,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:               true,
			Type:                 ExporterTypeStdout,
			Pretty:               true,
			IntervalSeconds:      10,
			EnableRuntimeMetrics: false,
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if provider.GetLoggerProvider() == nil {
		t.Error("log provider is nil")
	}

	if provider.GetTracerProvider() == nil {
		t.Error("trace provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Error("meter provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestNewProvider_DisabledProviders(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: false,
		},
		Trace: TraceConfig{
			Enable: false,
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if provider.GetLoggerProvider() != nil {
		t.Error("log provider should be nil when disabled")
	}

	if provider.GetTracerProvider() != nil {
		t.Error("trace provider should be nil when disabled")
	}

	if provider.GetMeterProvider() != nil {
		t.Error("meter provider should be nil when disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestNewProvider_NilConfig(t *testing.T) {
	provider, err := NewProvider(nil)
	if err != nil {
		t.Fatalf("failed to create provider with nil config: %v", err)
	}

	if provider == nil {
		t.Fatal("provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestShutdown_MultipleProviders(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First shutdown
	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("first shutdown failed: %v", err)
	}

	// Second shutdown should not error
	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("second shutdown failed: %v", err)
	}
}

func TestCreateLogExporter_InvalidType(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: true,
			Type:   "invalid",
		},
		Trace: TraceConfig{
			Enable: false,
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Should fallback to stdout
	if provider.GetLoggerProvider() == nil {
		t.Error("log provider should not be nil with invalid type")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}
