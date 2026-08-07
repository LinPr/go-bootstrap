package otel

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func TestNewProvider_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.ServiceVersion = "1.0.0"

	provider, err := newProvider(config)
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
			Enable:   true,
			Exporter: ExporterTypeStdout,
			Logger:   LoggerTypeSlog,
			Pretty:   true,
		},
		Trace: TraceConfig{
			Enable:        true,
			Exporter:      ExporterTypeStdout,
			Pretty:        true,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:               true,
			Exporter:             ExporterTypeStdout,
			Pretty:               true,
			IntervalSeconds:      10,
			EnableRuntimeMetrics: false,
		},
	}

	provider, err := newProvider(config)
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

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestNewProvider_NilConfig(t *testing.T) {
	provider, err := newProvider(nil)
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

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown should work
	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}

func TestShutdown_NoProviders(t *testing.T) {
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

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown with no providers should not error
	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}

func TestCreateLogExporter_InvalidType(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:   true,
			Exporter: "invalid",
			Logger:   LoggerTypeSlog,
		},
		Trace: TraceConfig{
			Enable: false,
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}

	_, err := newProvider(config)
	if err == nil {
		t.Error("expected error for invalid log exporter type")
	}
}

func TestCreateTraceExporter_InvalidType(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: false,
		},
		Trace: TraceConfig{
			Enable:   true,
			Exporter: "invalid",
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}

	_, err := newProvider(config)
	if err == nil {
		t.Error("expected error for invalid trace exporter type")
	}
}

func TestCreateMetricExporter_InvalidType(t *testing.T) {
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
			Enable:   true,
			Exporter: "invalid",
		},
	}

	_, err := newProvider(config)
	if err == nil {
		t.Error("expected error for invalid metric exporter type")
	}
}

func TestProvider_WithRuntimeMetrics(t *testing.T) {
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
			Enable:               true,
			Exporter:             ExporterTypeStdout,
			Pretty:               false,
			IntervalSeconds:      5,
			EnableRuntimeMetrics: true,
		},
	}

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider with runtime metrics: %v", err)
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

func TestProvider_DifferentSamplingRatios(t *testing.T) {
	tests := []struct {
		name          string
		samplingRatio float64
	}{
		{"100% sampling", 1.0},
		{"50% sampling", 0.5},
		{"10% sampling", 0.1},
		{"0% sampling", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable: false,
				},
				Trace: TraceConfig{
					Enable:        true,
					Exporter:      ExporterTypeStdout,
					Pretty:        false,
					SamplingRatio: tt.samplingRatio,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			}

			provider, err := newProvider(config)
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := provider.Shutdown(ctx); err != nil {
				t.Errorf("failed to shutdown provider: %v", err)
			}
		})
	}
}

func TestProvider_StdoutWithoutPretty(t *testing.T) {
	config := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:   true,
			Exporter: ExporterTypeStdout,
			Logger:   LoggerTypeSlog,
			Pretty:   false,
		},
		Trace: TraceConfig{
			Enable:        true,
			Exporter:      ExporterTypeStdout,
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:          true,
			Exporter:        ExporterTypeStdout,
			Pretty:          false,
			IntervalSeconds: 5,
		},
	}

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestProvider_PrometheusExporter(t *testing.T) {
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
			Enable:               true,
			Exporter:             ExporterTypePrometheus,
			EnableRuntimeMetrics: false,
		},
	}

	provider, err := newProvider(config)
	if err != nil {
		t.Fatalf("failed to create prometheus provider: %v", err)
	}

	if provider.GetMeterProvider() == nil {
		t.Error("meter provider is nil for prometheus")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown provider: %v", err)
	}
}

func TestProvider_LoggerBridges(t *testing.T) {
	tests := []struct {
		name       string
		loggerType LoggerType
	}{
		{
			name:       "Slog Logger",
			loggerType: LoggerTypeSlog,
		},
		{
			name:       "Zap Logger",
			loggerType: LoggerTypeZap,
		},
		{
			name:       "Logrus Logger",
			loggerType: LoggerTypeLogrus,
		},
		{
			name:       "Logr Logger",
			loggerType: LoggerTypeLogr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable:   true,
					Exporter: ExporterTypeStdout,
					Logger:   tt.loggerType,
					Pretty:   false,
				},
				Trace: TraceConfig{
					Enable: false,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			}

			provider, err := newProvider(config)
			if err != nil {
				t.Fatalf("failed to create provider with %s: %v", tt.loggerType, err)
			}

			if provider.GetLoggerProvider() == nil {
				t.Errorf("log provider is nil for %s", tt.loggerType)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := provider.Shutdown(ctx); err != nil {
				t.Errorf("failed to shutdown provider with %s: %v", tt.loggerType, err)
			}
		})
	}
}

func TestProvider_LoggerBridgesWithLogging(t *testing.T) {
	tests := []struct {
		name       string
		loggerType LoggerType
		logFunc    func()
	}{
		{
			name:       "Slog Logger Logging",
			loggerType: LoggerTypeSlog,
			logFunc: func() {
				slog.Info("test slog message", "key", "value")
				slog.Error("test slog error", "error", "test error")
			},
		},
		{
			name:       "Zap Logger Logging",
			loggerType: LoggerTypeZap,
			logFunc: func() {
				zap.L().Info("test zap message", zap.String("key", "value"))
				zap.L().Error("test zap error", zap.String("error", "test error"))
			},
		},
		{
			name:       "Logrus Logger Logging",
			loggerType: LoggerTypeLogrus,
			logFunc: func() {
				logrus.Info("test logrus message")
				logrus.WithField("key", "value").Info("test logrus with field")
				logrus.Error("test logrus error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable:   true,
					Exporter: ExporterTypeStdout,
					Logger:   tt.loggerType,
					Pretty:   true,
				},
				Trace: TraceConfig{
					Enable: false,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			}

			provider, err := newProvider(config)
			if err != nil {
				t.Fatalf("failed to create provider with %s: %v", tt.loggerType, err)
			}

			// 执行日志打印测试
			tt.logFunc()

			// 等待一下让日志输出
			time.Sleep(100 * time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := provider.Shutdown(ctx); err != nil {
				t.Errorf("failed to shutdown provider with %s: %v", tt.loggerType, err)
			}
		})
	}
}
