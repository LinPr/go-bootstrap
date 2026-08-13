package otel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/LinPr/go-bootstrap/otel/zapsugar"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogr"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type logrusState struct {
	output    io.Writer
	formatter logrus.Formatter
	level     logrus.Level
	hooks     logrus.LevelHooks
}

func snapshotLogrusState() logrusState {
	logger := logrus.StandardLogger()
	return logrusState{
		output:    logger.Out,
		formatter: logger.Formatter,
		level:     logger.Level,
		hooks:     logger.Hooks,
	}
}

func restoreLogrusState(state logrusState) {
	logger := logrus.StandardLogger()
	logger.Out = state.output
	logger.Formatter = state.formatter
	logger.Level = state.level
	logger.Hooks = state.hooks
}

func TestOtelProviderInitialization(t *testing.T) {
	previousSlog := slog.Default()
	previousZap := zap.L()
	previousLogrus := snapshotLogrusState()

	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
		zap.ReplaceGlobals(previousZap)
		restoreLogrusState(previousLogrus)
		globalProvider = nil
	})

	globalProvider = nil

	config := &Config{
		ServiceName:    "go-bootstrap-test",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
			Level:      "info",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-test",
			},
			Logger: LoggerTypeSlog,
			Pretty: false,
		},
		Trace: TraceConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/traces",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-test",
			},
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/metrics",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-test",
			},
			Pretty:          false,
			IntervalSeconds: 1,
		},
	}

	provider, err := NewOtelProviders(config)
	if err != nil {
		t.Fatalf("failed to initialize provider: %v", err)
	}

	if provider.GetTracerProvider() == nil {
		t.Fatal("tracer provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Fatal("meter provider is nil")
	}

	if provider.GetLoggerProvider() == nil {
		t.Fatal("logger provider is nil")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown provider: %v", err)
	}
}

func TestOtelLogger(t *testing.T) {
	previousSlog := slog.Default()
	previousZap := zap.L()
	previousLogrus := snapshotLogrusState()

	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
		zap.ReplaceGlobals(previousZap)
		restoreLogrusState(previousLogrus)
		globalProvider = nil
	})

	tests := []struct {
		name       string
		loggerType LoggerType
	}{
		{name: "slog", loggerType: LoggerTypeSlog},
		{name: "zap", loggerType: LoggerTypeZap},
		{name: "logrus", loggerType: LoggerTypeLogrus},
		{name: "logr", loggerType: LoggerTypeLogr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalProvider = nil

			config := newLoggerTestConfig(tt.loggerType)
			provider, err := NewOtelProviders(config)
			if err != nil {
				t.Fatalf("failed to initialize provider: %v", err)
			}

			runLoggerTest(t, tt.loggerType, provider)

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := ShutdownOtelProvider(shutdownCtx); err != nil {
				t.Fatalf("failed to shutdown provider: %v", err)
			}
		})
	}
}

func newLoggerTestConfig(loggerType LoggerType) *Config {
	return &Config{
		ServiceName:    "go-bootstrap",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
			Level:      "warn",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Logger: loggerType,
			Pretty: false,
		},
		Trace: TraceConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/traces",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
			},
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable: false,
		},
	}
}

func runLoggerTest(t *testing.T, loggerType LoggerType, provider *OtelProviders) {
	t.Helper()

	tracer := otel.Tracer("logger-test-tracer")
	rootCtx, rootSpan := tracer.Start(context.Background(), "logger-test-root")
	defer rootSpan.End()

	emitLogger(t, loggerType, provider, rootCtx, "root")

	time.Sleep(2 * time.Second)
}

func emitLogger(t *testing.T, loggerType LoggerType, provider *OtelProviders, ctx context.Context, phase string) {
	t.Helper()

	message := string(loggerType) + ": " + phase + " telemetry check"

	switch loggerType {
	case LoggerTypeSlog:
		slog.DebugContext(ctx, message+" debug", "phase", phase)
		slog.InfoContext(ctx, message+" info", "phase", phase)
		slog.WarnContext(ctx, message+" warn", "phase", phase)
		slog.ErrorContext(ctx, message+" error", "phase", phase)

		logger := slog.Default().WithGroup("sublogger")
		logger.DebugContext(ctx, message+" debug (sub)", "phase", phase)
		logger.InfoContext(ctx, message+" info (sub)", "phase", phase)
		logger.WarnContext(ctx, message+" warn (sub)", "phase", phase)
		logger.ErrorContext(ctx, message+" error (sub)", "phase", phase)

	case LoggerTypeZap:
		zap.L().Debug(message+" debug", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Info(message+" info", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Warn(message+" warn", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Error(message+" error", zap.Any("context", ctx), zap.String("phase", phase))

		zapsugar.Debugw(ctx, message+" debug (zapsugar)", "phase", phase)
		zapsugar.Infow(ctx, message+" info (zapsugar)", "phase", phase)
		zapsugar.Warnw(ctx, message+" warn (zapsugar)", "phase", phase)
		zapsugar.Errorw(ctx, message+" error (zapsugar)", "phase", phase)

		subLogger := zapsugar.NewSubScopedZapSugar("sublogger", nil)
		subLogger.Infow(ctx, message+" info (sub)", "phase", phase)
		subLogger.Warnw(ctx, message+" warn (sub)", "phase", phase)

	case LoggerTypeLogrus:
		logrus.WithContext(ctx).WithField("phase", phase).Debug(message + " debug")
		logrus.WithContext(ctx).WithField("phase", phase).Info(message + " info")
		logrus.WithContext(ctx).WithField("phase", phase).Warn(message + " warn")
		logrus.WithContext(ctx).WithField("phase", phase).Error(message + " error")

		subLogger := logrus.WithField("module", "sublogger")
		subLogger.WithContext(ctx).WithField("phase", phase).Info(message + " info (sub)")
		subLogger.WithContext(ctx).WithField("phase", phase).Warn(message + " warn (sub)")

	case LoggerTypeLogr:
		logger := logr.New(otellogr.NewLogSink("integration", otellogr.WithLoggerProvider(provider.GetLoggerProvider())))
		logger.WithValues("context", ctx, "phase", phase).V(1).Info(message + " debug")
		logger.WithValues("context", ctx, "phase", phase).Info(message + " info")
		logger.WithValues("context", ctx, "phase", phase).Error(fmt.Errorf("synthetic error"), message+" error")
	default:
		t.Fatalf("unsupported logger type: %s", loggerType)
	}
}
