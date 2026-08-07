package otel

import "testing"

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ServiceName != "default-service" {
		t.Errorf("expected service name 'default-service', got '%s'", config.ServiceName)
	}

	if config.ServiceVersion != "0.0.0" {
		t.Errorf("expected service version '0.0.0', got '%s'", config.ServiceVersion)
	}

	if !config.Log.Enable {
		t.Error("log should be enabled by default")
	}

	if config.Log.Exporter != ExporterTypeStdout {
		t.Errorf("expected log type '%s', got '%s'", ExporterTypeStdout, config.Log.Exporter)
	}

	if !config.Trace.Enable {
		t.Error("trace should be enabled by default")
	}

	if config.Trace.SamplingRatio != 1.0 {
		t.Errorf("expected sampling ratio 1.0, got %f", config.Trace.SamplingRatio)
	}

	if !config.Metric.Enable {
		t.Error("metric should be enabled by default")
	}

	if config.Metric.IntervalSeconds != 10 {
		t.Errorf("expected interval 10 seconds, got %d", config.Metric.IntervalSeconds)
	}

	if !config.Metric.EnableRuntimeMetrics {
		t.Error("runtime metrics should be enabled by default")
	}
}

func TestExporterTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		expected ExporterType
		value    string
	}{
		{"stdout", ExporterTypeStdout, "stdout"},
		{"http", ExporterTypeHTTP, "http"},
		{"grpc", ExporterTypeGRPC, "grpc"},
		{"prometheus", ExporterTypePrometheus, "prometheus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.expected) != tt.value {
				t.Errorf("expected %s, got %s", tt.value, tt.expected)
			}
		})
	}
}

func TestConfigModification(t *testing.T) {
	config := DefaultConfig()

	// Modify config
	config.ServiceName = "my-service"
	config.ServiceVersion = "1.2.3"
	config.Log.Exporter = ExporterTypeHTTP
	config.Log.RemoteAddr = "http://localhost:4318"
	config.Trace.Exporter = ExporterTypeGRPC
	config.Trace.RemoteAddr = "localhost:4317"
	config.Metric.Exporter = ExporterTypePrometheus

	// Verify modifications
	if config.ServiceName != "my-service" {
		t.Errorf("failed to modify service name")
	}

	if config.ServiceVersion != "1.2.3" {
		t.Errorf("failed to modify service version")
	}

	if config.Log.Exporter != ExporterTypeHTTP {
		t.Errorf("failed to modify log type")
	}

	if config.Trace.Exporter != ExporterTypeGRPC {
		t.Errorf("failed to modify trace type")
	}

	if config.Metric.Exporter != ExporterTypePrometheus {
		t.Errorf("failed to modify metric type")
	}
}
