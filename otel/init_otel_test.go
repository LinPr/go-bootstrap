package otel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LinPr/go-bootstrap/otel/transport"
	"github.com/LinPr/go-bootstrap/otel/zapsugar"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogr"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
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

func TestOtelMetrics(t *testing.T) {
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

	config := newMetricsTestConfig()
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

	runMetricsTest(t, provider)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown provider: %v", err)
	}
}

func runMetricsTest(t *testing.T, provider *OtelProviders) {
	t.Helper()

	meter := otel.Meter("go-bootstrap")
	metricPrefix := "go_bootstrap_"
	requestCounter, err := meter.Int64Counter(metricPrefix + "request_counter")
	if err != nil {
		t.Fatalf("failed to create request counter: %v", err)
	}
	histogram, err := meter.Float64Histogram(metricPrefix + "latency_ms")
	if err != nil {
		t.Fatalf("failed to create latency histogram: %v", err)
	}
	activeCounter, err := meter.Int64UpDownCounter(metricPrefix + "active_requests")
	if err != nil {
		t.Fatalf("failed to create active counter: %v", err)
	}

	tracer := otel.Tracer("metrics-test-tracer")

	var httpServerTraceID, grpcServerTraceID string
	httpServer := httptest.NewServer(otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCtx := r.Context()
		httpServerTraceID = trace.SpanContextFromContext(serverCtx).TraceID().String()
		slog.InfoContext(serverCtx, "HTTP server received request", "path", r.URL.Path, "method", r.Method)
		requestCounter.Add(serverCtx, 1, metric.WithAttributes(attribute.String("transport", "http-server")))
		histogram.Record(serverCtx, 12.5, metric.WithAttributes(attribute.String("transport", "http-server")))
		_, _ = w.Write([]byte("ok"))
	}), "http-server"))
	defer httpServer.Close()

	httpClient := &http.Client{Transport: transport.NewOtelHttpTransport()}

	grpcUnaryHandler := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		grpcServerTraceID = trace.SpanContextFromContext(ctx).TraceID().String()
		slog.InfoContext(ctx, "gRPC server received request", "method", info.FullMethod)
		requestCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		histogram.Record(ctx, 21.0, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		activeCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		defer activeCounter.Add(ctx, -1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		return handler(ctx, req)
	}

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer(
		transport.WithOtelGRPCServerOption(),
		grpc.UnaryInterceptor(grpcUnaryHandler),
	)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("metrics.Service", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		transport.WithOtelGRPCClientOption(),
	)
	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	grpcClient := healthpb.NewHealthClient(conn)

	const iterations = 5
	for i := 0; i < iterations; i++ {
		rootCtx, rootSpan := tracer.Start(context.Background(), fmt.Sprintf("metrics-root-%d", i))
		rootTraceID := rootSpan.SpanContext().TraceID().String()

		slog.InfoContext(rootCtx, "Starting metrics test iteration", "iteration", i, "trace_id", rootTraceID)

		httpCtx, httpSpan := tracer.Start(rootCtx, "http-client-call")
		httpRequest, err := http.NewRequestWithContext(httpCtx, http.MethodGet, httpServer.URL+"/ping", nil)
		if err != nil {
			t.Fatalf("iteration %d: failed to create http request: %v", i, err)
		}
		slog.InfoContext(httpCtx, "Sending HTTP request", "iteration", i, "url", httpServer.URL+"/ping")
		httpResponse, err := httpClient.Do(httpRequest)
		if err != nil {
			t.Fatalf("iteration %d: http request failed: %v", i, err)
		}
		_ = httpResponse.Body.Close()
		if httpResponse.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: unexpected http status: %d", i, httpResponse.StatusCode)
		}
		requestCounter.Add(httpCtx, 1, metric.WithAttributes(attribute.String("transport", "http-client")))
		histogram.Record(httpCtx, 8.25, metric.WithAttributes(attribute.String("transport", "http-client")))
		httpSpan.End()

		grpcCtx, grpcSpan := tracer.Start(rootCtx, "grpc-client-call")
		slog.InfoContext(grpcCtx, "Sending gRPC health check", "iteration", i, "service", "metrics.Service")
		resp, err := grpcClient.Check(grpcCtx, &healthpb.HealthCheckRequest{Service: "metrics.Service"})
		if err != nil {
			t.Fatalf("iteration %d: grpc health check failed: %v", i, err)
		}
		slog.Info("rpc check resp: " + resp.String())
		resp.GetStatus()
		requestCounter.Add(grpcCtx, 1, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		histogram.Record(grpcCtx, 34.0, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		grpcSpan.End()

		requestCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		histogram.Record(rootCtx, 5.0, metric.WithAttributes(attribute.String("transport", "root")))
		activeCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		activeCounter.Add(rootCtx, -1, metric.WithAttributes(attribute.String("transport", "root")))

		rootSpan.End()
		slog.InfoContext(rootCtx, "Completed metrics test iteration", "iteration", i)

		if i == 0 {
			if httpServerTraceID == "" {
				t.Fatal("http server trace id is empty")
			}
			if grpcServerTraceID == "" {
				t.Fatal("grpc server trace id is empty")
			}
			if httpServerTraceID != rootTraceID {
				t.Fatalf("expected http trace id %s, got %s", rootTraceID, httpServerTraceID)
			}
			if grpcServerTraceID != rootTraceID {
				t.Fatalf("expected grpc trace id %s, got %s", rootTraceID, grpcServerTraceID)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	metricsServer := httptest.NewServer(promhttp.Handler())
	defer metricsServer.Close()

	metricsResp, err := http.Get(metricsServer.URL)
	if err != nil {
		t.Fatalf("failed to scrape prometheus metrics: %v", err)
	}
	defer metricsResp.Body.Close()
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

func newMetricsTestConfig() *Config {
	return &Config{
		ServiceName:    "go-bootstrap",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
			Level:      "info",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap",
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
				"stream-name":   "go-bootstrap",
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
				"stream-name":   "go-bootstrap",
			},
			Pretty:          false,
			IntervalSeconds: 1,
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
		slog.Default().Handler()
		logger.DebugContext(ctx, message+" debug (sub)", "phase", phase)
		logger.InfoContext(ctx, message+" info (sub)", "phase", phase)
		logger.WarnContext(ctx, message+" warn (sub)", "phase", phase)
		logger.ErrorContext(ctx, message+" error (sub)", "phase", phase)

	case LoggerTypeZap:
		zap.L().Debug(message+" debug", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Info(message+" info", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Warn(message+" warn", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Error(message+" error", zap.Any("context", ctx), zap.String("phase", phase))

		zap.S().Debugw(message+" debug (sugar)", "context", ctx, "phase", phase)
		zap.S().Infow(message+" info (sugar)", "context", ctx, "phase", phase)
		zap.S().Warnw(message+" warn (sugar)", "context", ctx, "phase", phase)
		zap.S().Errorw(message+" error (sugar)", "context", ctx, "phase", phase)

		zapsugar.Debugw(ctx, message+" debug (zapsugar)", "phase", phase)
		zapsugar.Infow(ctx, message+" info (zapsugar)", "phase", phase)
		zapsugar.Warnw(ctx, message+" warn (zapsugar)", "phase", phase)
		zapsugar.Errorw(ctx, message+" error (zapsugar)", "phase", phase)

		zapsugar.Debugf(ctx, "%s debug (zapsugar-f) phase=%s", message, phase)
		zapsugar.Infof(ctx, "%s info (zapsugar-f) phase=%s", message, phase)
		zapsugar.Warnf(ctx, "%s warn (zapsugar-f) phase=%s", message, phase)
		zapsugar.Errorf(ctx, "%s error (zapsugar-f) phase=%s", message, phase)

		subLogger := zapsugar.NewSubScopedZapSugar("sublogger", nil)
		subLogger.Debugw(ctx, message+" debug (sub)", "phase", phase)
		subLogger.Infow(ctx, message+" info (sub)", "phase", phase)
		subLogger.Warnw(ctx, message+" warn (sub)", "phase", phase)
		subLogger.Errorw(ctx, message+" error (sub)", "phase", phase)

		subLogger.Debugf(ctx, "%s debug (sub-f) phase=%s", message, phase)
		subLogger.Infof(ctx, "%s info (sub-f) phase=%s", message, phase)
		subLogger.Warnf(ctx, "%s warn (sub-f) phase=%s", message, phase)
		subLogger.Errorf(ctx, "%s error (sub-f) phase=%s", message, phase)

		nestedLogger := zapsugar.NewSubScopedZapSugar("nested", subLogger)
		nestedLogger.Debugw(ctx, message+" debug (nested)", "phase", phase)
		nestedLogger.Infow(ctx, message+" info (nested)", "phase", phase)
		nestedLogger.Warnw(ctx, message+" warn (nested)", "phase", phase)
		nestedLogger.Errorw(ctx, message+" error (nested)", "phase", phase)

		// nestedLogger.Debugf(ctx, "%s debug (nested-f) phase=%s", message, phase)
		// nestedLogger.Infof(ctx, "%s info (nested-f) phase=%s", message, phase)
		// nestedLogger.Warnf(ctx, "%s warn (nested-f) phase=%s", message, phase)
		// nestedLogger.Errorf(ctx, "%s error (nested-f) phase=%s", message, phase)

		attributedLogger := subLogger.WithAttribute("request_id", "req-12345")
		attributedLogger.Debugw(ctx, message+" debug (attributed)", "phase", phase)
		attributedLogger.Infow(ctx, message+" info (attributed)", "phase", phase)
		attributedLogger.Warnw(ctx, message+" warn (attributed)", "phase", phase)
		attributedLogger.Errorw(ctx, message+" error (attributed)", "phase", phase)

		// attributedLogger.Debugf(ctx, "%s debug (attributed-f) phase=%s", message, phase)
		// attributedLogger.Infof(ctx, "%s info (attributed-f) phase=%s", message, phase)
		// attributedLogger.Warnf(ctx, "%s warn (attributed-f) phase=%s", message, phase)
		// attributedLogger.Errorf(ctx, "%s error (attributed-f) phase=%s", message, phase)

	case LoggerTypeLogrus:
		logrus.WithContext(ctx).WithField("phase", phase).Debug(message + " debug")
		logrus.WithContext(ctx).WithField("phase", phase).Info(message + " info")
		logrus.WithContext(ctx).WithField("phase", phase).Warn(message + " warn")
		logrus.WithContext(ctx).WithField("phase", phase).Error(message + " error")

		subLogger := logrus.WithField("module", "sublogger")
		subLogger.WithContext(ctx).WithField("phase", phase).Debug(message + " debug (sub)")
		subLogger.WithContext(ctx).WithField("phase", phase).Info(message + " info (sub)")
		subLogger.WithContext(ctx).WithField("phase", phase).Warn(message + " warn (sub)")
		subLogger.WithContext(ctx).WithField("phase", phase).Error(message + " error (sub)")

		nestedLogger := subLogger.WithField("component", "nested")
		nestedLogger.WithContext(ctx).WithField("phase", phase).Debug(message + " debug (nested)")
		nestedLogger.WithContext(ctx).WithField("phase", phase).Info(message + " info (nested)")
		nestedLogger.WithContext(ctx).WithField("phase", phase).Warn(message + " warn (nested)")
		nestedLogger.WithContext(ctx).WithField("phase", phase).Error(message + " error (nested)")

	case LoggerTypeLogr:
		logger := logr.New(otellogr.NewLogSink("integration", otellogr.WithLoggerProvider(provider.GetLoggerProvider())))
		logger.WithValues("context", ctx, "phase", phase).V(1).Info(message + " debug")
		logger.WithValues("context", ctx, "phase", phase).Info(message + " info")
		logger.WithValues("context", ctx, "phase", phase).Error(fmt.Errorf("synthetic error"), message+" error")
	default:
		t.Fatalf("unsupported logger type: %s", loggerType)
	}
}
