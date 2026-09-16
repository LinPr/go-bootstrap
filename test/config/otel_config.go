package config

import (
	"log/slog"
	"testing"

	"github.com/LinPr/go-bootstrap/otel"
	"github.com/LinPr/go-bootstrap/otel/slogbaggage"
)

// func NewTestOtelConfig() *otel.Config {
// 	return &otel.Config{
// 		ServiceName:    "go-bootstrap",
// 		ServiceVersion: "1.0.0",
// 		Log: otel.LogConfig{
// 			Enable:     true,
// 			Exporter:   otel.ExporterTypeHTTP,
// 			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
// 			Level:      "info",
// 			Headers: map[string]string{
// 				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
// 				"stream-name":   "go-bootstrap",
// 			},
// 			Logger: otel.LoggerTypeSlog,
// 			Pretty: true,
// 		},
// 		Trace: otel.TraceConfig{
// 			Enable:     true,
// 			Exporter:   otel.ExporterTypeHTTP,
// 			RemoteAddr: "http://10.86.11.34:5318/v1/traces",
// 			// RemoteAddr: "http://10.86.11.69:9428/insert/opentelemetry/v1/traces",
// 			Headers: map[string]string{
// 				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
// 				"stream-name":   "go-bootstrap",
// 			},
// 			Pretty:        false,
// 			SamplingRatio: 1.0,
// 		},
// 		Metric: otel.MetricConfig{
// 			Enable:     true,
// 			Exporter:   otel.ExporterTypeHTTP,
// 			RemoteAddr: "http://10.86.11.34:5318/v1/metrics",
// 			Headers: map[string]string{
// 				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
// 				"stream-name":   "go-bootstrap",
// 			},
// 			Pretty:          false,
// 			IntervalSeconds: 1,
// 		},
// 	}
// }

func NewTestOtelConfig() *otel.Config {
	return &otel.Config{
		ServiceName:    "go-bootstrap",
		ServiceVersion: "1.0.0",
		Log: otel.LogConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/logs",
			Level:      "debug",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Logger:              otel.LoggerTypeSlog,
			Pretty:              true,
			AttributeCountLimit: 128,
		},
		Trace: otel.TraceConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/traces",
			// RemoteAddr: "http://10.86.11.69:9428/insert/opentelemetry/v1/traces",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: otel.MetricConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/metrics",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Pretty:               false,
			IntervalSeconds:      1,
			EnableRuntimeMetrics: true,
			CardinalityLimit:     1000,
		},
	}
}

func SetupTestOtelProvider(t *testing.T) *otel.OtelProviders {
	previousSlog := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
	})

	config := NewTestOtelConfig()
	provider, err := otel.NewOtelProviders(config)
	if err != nil {
		t.Fatalf("failed to initialize provider: %v", err)
	}

	if provider.GetTracerProvider() == nil {
		t.Fatal("tracer provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Fatal("meter provider is nil")
	}

	logger := slog.New(
		slogbaggage.NewBaggageHandler(
			slog.Default().Handler(),
			slogbaggage.WithBaggageMembers(
				"BaggageKey",
			),
		),
	)
	slog.SetDefault(logger)

	return provider
}

// SetupFilelogProvider initializes an OTel provider configured for file log
// output with lumberjack rotation. It mirrors NewTestOtelConfig for the
// Trace section but switches the Log exporter to ExporterTypeFile with the
// given Rotate settings. The returned provider has trace enabled so that log
// records carry trace context.
func SetupFilelogProvider(t *testing.T, rotate otel.Rotate) *otel.OtelProviders {
	previousSlog := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
	})

	config := &otel.Config{
		ServiceName:    "go-bootstrap",
		ServiceVersion: "1.0.0",
		Log: otel.LogConfig{
			Enable:   true,
			Exporter: otel.ExporterTypeFile,
			Logger:   otel.LoggerTypeSlog,
			Level:    "debug",
			Pretty:   false,
			Rotate:   rotate,
			// AttributeCountLimit: 128,
		},
		Trace: otel.TraceConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/traces",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: otel.MetricConfig{
			Enable:     false,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/metrics",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Pretty:               false,
			IntervalSeconds:      1,
			EnableRuntimeMetrics: true,
			// CardinalityLimit:     2000,
		},
	}

	provider, err := otel.NewOtelProviders(config)
	if err != nil {
		t.Fatalf("failed to initialize provider: %v", err)
	}

	if provider.GetTracerProvider() == nil {
		t.Fatal("tracer provider is nil")
	}

	return provider
}
