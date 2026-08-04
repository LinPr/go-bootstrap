package gobootstrap

import (
	"context"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
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

	err := Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if globalProvider == nil {
		t.Fatal("global provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	// Reset global provider for other tests
	globalProvider = nil
}

func TestGetGlobalProvider(t *testing.T) {
	// Initially should be nil
	if GetGlobalProvider() != nil {
		t.Error("global provider should be nil initially")
	}

	config := DefaultConfig()
	config.ServiceName = "test-service"

	if err := Initialize(config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	provider := GetGlobalProvider()
	if provider == nil {
		t.Fatal("global provider is nil after initialization")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

func TestShutdown_NoProvider(t *testing.T) {
	globalProvider = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not error when no provider is set
	if err := Shutdown(ctx); err != nil {
		t.Errorf("shutdown should not error when no provider is set: %v", err)
	}
}

func TestGetProviders(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"

	if err := Initialize(config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if GetLoggerProvider() == nil {
		t.Error("logger provider is nil")
	}

	if GetTracerProvider() == nil {
		t.Error("tracer provider is nil")
	}

	if GetMeterProvider() == nil {
		t.Error("meter provider is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}
