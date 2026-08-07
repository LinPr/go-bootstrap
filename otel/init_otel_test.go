package otel

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

func TestInitOtelProvider(t *testing.T) {
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
			Enable:               true,
			Exporter:             ExporterTypeStdout,
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
					Enable:   true,
					Exporter: ExporterTypeStdout,
					Logger:   LoggerTypeSlog,
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
					Exporter:      ExporterTypeStdout,
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
					Exporter:             ExporterTypeStdout,
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

func TestInitOtelProvider_LoggerBridges(t *testing.T) {
	tests := []struct {
		name       string
		loggerType LoggerType
		testFunc   func(t *testing.T)
	}{
		{
			name:       "Slog Logger Bridge",
			loggerType: LoggerTypeSlog,
			testFunc: func(t *testing.T) {
				// 测试 slog 日志打印
				slog.Info("test slog info message", "test_key", "test_value", "number", 42)
				slog.Warn("test slog warn message", "warning", "test warning")
				slog.Error("test slog error message", "error", "test error")

				t.Log("Slog logger: printed info, warn, and error messages")
			},
		},
		{
			name:       "Zap Logger Bridge",
			loggerType: LoggerTypeZap,
			testFunc: func(t *testing.T) {
				// 测试 zap 日志打印
				zap.L().Info("test zap info message",
					zap.String("test_key", "test_value"),
					zap.Int("number", 42))
				zap.L().Warn("test zap warn message",
					zap.String("warning", "test warning"))
				zap.L().Error("test zap error message",
					zap.String("error", "test error"))

				t.Log("Zap logger: printed info, warn, and error messages")
			},
		},
		{
			name:       "Logrus Logger Bridge",
			loggerType: LoggerTypeLogrus,
			testFunc: func(t *testing.T) {
				// 测试 logrus 日志打印
				logrus.Info("test logrus info message")
				logrus.WithFields(logrus.Fields{
					"test_key": "test_value",
					"number":   42,
				}).Info("test logrus info with fields")
				logrus.Warn("test logrus warn message")
				logrus.WithField("warning", "test warning").Warn("test logrus warn with field")
				logrus.Error("test logrus error message")
				logrus.WithField("error", "test error").Error("test logrus error with field")

				t.Log("Logrus logger: printed info, warn, and error messages")
			},
		},
		{
			name:       "Logr Logger Bridge",
			loggerType: LoggerTypeLogr,
			testFunc: func(t *testing.T) {
				// logr 需要用户自己管理实例，这里只验证初始化不报错
				t.Log("Logr logger: initialized successfully (requires manual instance management)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalProvider = nil

			config := &Config{
				ServiceName:    string(tt.loggerType),
				ServiceVersion: "1.0.0",
				Log: LogConfig{
					Enable:     true,
					Exporter:   ExporterTypeHTTP,
					RemoteAddr: "http://10.86.11.34:5318/v1/logs",
					Headers: map[string]string{
						"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
						"stream-name":   "vnet-bff-dev",
					},
					Logger: tt.loggerType,
					Pretty: false,
				},
				Trace: TraceConfig{
					Enable: false,
				},
				Metric: MetricConfig{
					Enable: false,
				},
			}

			err := InitOtelProvider(config)
			if err != nil {
				t.Fatalf("failed to initialize with %s: %v", tt.loggerType, err)
			}

			provider := GetOtelProvider()
			if provider == nil {
				t.Fatal("global provider is nil")
			}

			if provider.GetLoggerProvider() == nil {
				t.Errorf("logger provider is nil for %s", tt.loggerType)
			}

			// 执行实际的日志打印测试
			tt.testFunc(t)

			// 等待日志输出和发送
			time.Sleep(200 * time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := ShutdownOtelProvider(ctx); err != nil {
				t.Errorf("failed to shutdown with %s: %v", tt.loggerType, err)
			}

			globalProvider = nil
		})
	}
}

func TestInitOtelProvider_TraceWithSpans(t *testing.T) {
	globalProvider = nil

	config := &Config{
		ServiceName:    "test-trace-service",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable: false,
		},
		Trace: TraceConfig{
			Enable:        true,
			Exporter:      ExporterTypeStdout,
			Pretty:        true,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}

	err := InitOtelProvider(config)
	if err != nil {
		t.Fatalf("failed to initialize trace provider: %v", err)
	}

	provider := GetOtelProvider()
	if provider == nil {
		t.Fatal("global provider is nil")
	}

	if provider.GetTracerProvider() == nil {
		t.Fatal("tracer provider is nil")
	}

	// 创建并使用 tracer
	tracer := otel.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-operation")
	span.SetAttributes(
		attribute.String("test.key", "test.value"),
		attribute.Int("test.number", 42),
	)

	// 模拟一些工作
	time.Sleep(10 * time.Millisecond)

	// 创建子 span
	_, childSpan := tracer.Start(ctx, "child-operation")
	childSpan.SetAttributes(attribute.String("child.key", "child.value"))
	time.Sleep(5 * time.Millisecond)
	childSpan.End()

	span.End()

	t.Log("Trace: created and ended spans with attributes")

	// 等待 span 输出
	time.Sleep(200 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

func TestInitOtelProvider_MetricWithInstruments(t *testing.T) {
	globalProvider = nil

	config := &Config{
		ServiceName:    "test-metric-service",
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
			Pretty:               true,
			IntervalSeconds:      1,
			EnableRuntimeMetrics: false,
		},
	}

	err := InitOtelProvider(config)
	if err != nil {
		t.Fatalf("failed to initialize metric provider: %v", err)
	}

	provider := GetOtelProvider()
	if provider == nil {
		t.Fatal("global provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Fatal("meter provider is nil")
	}

	// 创建并使用 meter
	meter := otel.Meter("test-meter")

	// 创建 Counter
	counter, err := meter.Int64Counter(
		"test.counter",
		metric.WithDescription("A test counter"),
	)
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	// 创建 Histogram
	histogram, err := meter.Float64Histogram(
		"test.histogram",
		metric.WithDescription("A test histogram"),
	)
	if err != nil {
		t.Fatalf("failed to create histogram: %v", err)
	}

	// 创建 UpDownCounter
	upDownCounter, err := meter.Int64UpDownCounter(
		"test.updowncounter",
		metric.WithDescription("A test up-down counter"),
	)
	if err != nil {
		t.Fatalf("failed to create up-down counter: %v", err)
	}

	ctx := context.Background()

	// 记录一些指标
	for i := 0; i < 5; i++ {
		counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", "GET"),
			attribute.String("status", "200"),
		))

		histogram.Record(ctx, float64(i*10+50), metric.WithAttributes(
			attribute.String("endpoint", "/api/test"),
		))

		upDownCounter.Add(ctx, int64(i%2*2-1), metric.WithAttributes(
			attribute.String("resource", "connection"),
		))

		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Metric: recorded counter, histogram, and up-down counter values")

	// 等待指标收集和输出
	time.Sleep(2 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

func TestInitOtelProvider_AllProvidersTogether(t *testing.T) {
	globalProvider = nil

	config := &Config{
		ServiceName:    "test-all-providers",
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
			IntervalSeconds:      1,
			EnableRuntimeMetrics: true,
		},
	}

	err := InitOtelProvider(config)
	if err != nil {
		t.Fatalf("failed to initialize all providers: %v", err)
	}

	provider := GetOtelProvider()
	if provider == nil {
		t.Fatal("global provider is nil")
	}

	// 验证所有 provider 都已初始化
	if provider.GetLoggerProvider() == nil {
		t.Error("logger provider is nil")
	}
	if provider.GetTracerProvider() == nil {
		t.Error("tracer provider is nil")
	}
	if provider.GetMeterProvider() == nil {
		t.Error("meter provider is nil")
	}

	ctx := context.Background()

	// 同时使用所有三个 provider
	// 1. Log
	slog.Info("testing all providers together", "test_id", "all-providers-test")

	// 2. Trace
	tracer := otel.Tracer("test-all-tracer")
	ctx, span := tracer.Start(ctx, "all-providers-operation")
	span.SetAttributes(attribute.String("operation", "test-all"))

	// 3. Metric
	meter := otel.Meter("test-all-meter")
	counter, _ := meter.Int64Counter("test.all.counter")
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", "all")))

	time.Sleep(100 * time.Millisecond)
	span.End()

	t.Log("All providers: log, trace, and metric working together")

	// 等待所有输出
	time.Sleep(2 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Errorf("failed to shutdown: %v", err)
	}

	globalProvider = nil
}

