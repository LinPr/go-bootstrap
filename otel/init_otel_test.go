package otel

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/LinPr/go-bootstrap/otel/zapsugar"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.uber.org/zap"
)

// type logrusState struct {
// 	output    io.Writer
// 	formatter logrus.Formatter
// 	level     logrus.Level
// 	hooks     logrus.LevelHooks
// }

// func snapshotLogrusState() logrusState {
// 	logger := logrus.StandardLogger()
// 	return logrusState{
// 		output:    logger.Out,
// 		formatter: logger.Formatter,
// 		level:     logger.Level,
// 		hooks:     logger.Hooks,
// 	}
// }

// func restoreLogrusState(state logrusState) {
// 	logger := logrus.StandardLogger()
// 	logger.Out = state.output
// 	logger.Formatter = state.formatter
// 	logger.Level = state.level
// 	logger.Hooks = state.hooks
// }

func TestOtelLogger(t *testing.T) {
	previousSlog := slog.Default()
	previousZap := zap.L()
	// previousLogrus := snapshotLogrusState()

	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
		zap.ReplaceGlobals(previousZap)
		// restoreLogrusState(previousLogrus)
		globalProvider = nil
	})

	tests := []struct {
		name       string
		loggerType LoggerType
	}{
		{name: "slog", loggerType: LoggerTypeSlog},
		// {name: "zap", loggerType: LoggerTypeZap},
		// {name: "logrus", loggerType: LoggerTypeLogrus},
		// {name: "logr", loggerType: LoggerTypeLogr},
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

			shutdownCtx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
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
			Level:      "error",
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
	rootCtx, rootSpan := tracer.Start(t.Context(), "logger-test-root")
	defer rootSpan.End()

	emitLogger(t, loggerType, provider, rootCtx, "root")

	time.Sleep(2 * time.Second)
}

func emitLogger(t *testing.T, loggerType LoggerType, provider *OtelProviders, ctx context.Context, phase string) {
	t.Helper()

	message := string(loggerType) + ": " + phase + " telemetry check"
	type Person struct {
		Name   string
		Age    int
		gemder string
	}
	person := Person{Name: "Alice", Age: 30, gemder: "Female"}
	switch loggerType {
	case LoggerTypeSlog:
		slog.DebugContext(ctx, message+" debug", "phase", phase)
		slog.InfoContext(ctx, message+" info", "phase", phase)
		slog.WarnContext(ctx, message+" warn", "phase", phase)
		slog.ErrorContext(ctx, message+" error", "phase", phase)

		slog.Debug(message+" debug (person)", "phase", phase, "person", person)
		slog.Info(message+" info (person)", "phase", phase, "person", person)
		slog.Warn(message+" warn (person)", "phase", phase, "person", person)
		slog.Error(message+" error (person)", "phase", phase, "person", person)

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

		zapsugar.Debugw(ctx, message+" debug (person)", "phase", phase, zap.Any("person", person))
		zapsugar.Infow(ctx, message+" info (person)", "phase", phase, zap.Any("person", person))
		zapsugar.Warnw(ctx, message+" warn (person)", "phase", phase, zap.Any("person", person))
		zapsugar.Errorw(ctx, message+" error (person)", "phase", phase, zap.Any("person", person))

		subLogger := zapsugar.NewSubScopedZapSugar("sublogger", zap.L().Sugar())
		subLogger.Infow(ctx, message+" info (sublogger)", "phase", phase)
		subLogger.Warnw(ctx, message+" warn (sublogger)", "phase", phase)
		subLogger.Errorw(ctx, message+" warn (sublogger)", "phase", phase)

		baggageLogger := zapsugar.NewSubScopedZapSugar("baggage-test", subLogger.Logger()).
			WithBaggageMembers("user.id", "request.id")
		ctxWithBaggage := addBaggageToCtx(ctx, "user.id", "12345", "request.id", "req-789", "session.id", "sess-ignored")
		baggageLogger.Warnw(ctxWithBaggage, message+" info (baggageLogger)", "phase", phase)
		baggageLogger.Errorw(ctxWithBaggage, message+" warn (baggageLogger)", "phase", phase)

		zap.ReplaceGlobals(baggageLogger.Logger().Desugar())
		zapsugar.Warnw(ctxWithBaggage, message+" debug (ReplaceGlobals)", "phase", phase)
		zapsugar.Errorw(ctxWithBaggage, message+" info (ReplaceGlobals)", "phase", phase)

	case LoggerTypeLogrus:
		// logrus.WithContext(ctx).WithField("phase", phase).Debug(message + " debug")
		// logrus.WithContext(ctx).WithField("phase", phase).Info(message + " info")
		// logrus.WithContext(ctx).WithField("phase", phase).Warn(message + " warn")
		// logrus.WithContext(ctx).WithField("phase", phase).Error(message + " error")

		// subLogger := logrus.WithField("module", "sublogger")
		// subLogger.WithContext(ctx).WithField("phase", phase).Info(message + " info (sub)")
		// subLogger.WithContext(ctx).WithField("phase", phase).Warn(message + " warn (sub)")

	case LoggerTypeLogr:
		// logger := logr.New(otellogr.NewLogSink("integration", otellogr.WithLoggerProvider(provider.GetLoggerProvider())))
		// logger.WithValues("context", ctx, "phase", phase).V(1).Info(message + " debug")
		// logger.WithValues("context", ctx, "phase", phase).Info(message + " info")
		// logger.WithValues("context", ctx, "phase", phase).Error(fmt.Errorf("synthetic error"), message+" error")
	default:
		t.Fatalf("unsupported logger type: %s", loggerType)
	}
}

func addBaggageToCtx(ctx context.Context, pairs ...string) context.Context {
	members := make([]baggage.Member, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m, _ := baggage.NewMember(pairs[i], pairs[i+1])
		members = append(members, m)
	}
	bag, _ := baggage.New(members...)
	return baggage.ContextWithBaggage(ctx, bag)
}
