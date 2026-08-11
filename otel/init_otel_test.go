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

func TestInitOtelProvider_TraceWithSpans(t *testing.T) {
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
			runInitOtelProviderIntegration(t, tt.loggerType)
		})
	}
}

func runInitOtelProviderIntegration(t *testing.T, loggerType LoggerType) {
	t.Helper()

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
		ServiceName:    "integration-" + string(loggerType),
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
			Level:      "error",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "test",
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
				"stream-name":   "test",
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
				"stream-name":   "test",
			},
			Pretty:          false,
			IntervalSeconds: 1,
		},
	}

	provider, err := NewOtelProviders(config)
	if err != nil {
		t.Fatalf("failed to initialize provider: %v", err)
	}

	if provider == nil {
		t.Fatal("provider is nil")
	}

	if GetOtelProvider() == nil {
		t.Fatal("global provider is nil")
	}

	if provider.GetLoggerProvider() == nil {
		t.Fatal("logger provider is nil")
	}

	if provider.GetTracerProvider() == nil {
		t.Fatal("tracer provider is nil")
	}

	if provider.GetMeterProvider() == nil {
		t.Fatal("meter provider is nil")
	}

	meter := otel.Meter("integration-meter")
	metricPrefix := "integration_otel_" + string(loggerType) + "_"
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

	tracer := otel.Tracer("integration-tracer")
	rootCtx, rootSpan := tracer.Start(context.Background(), "integration-root")
	rootTraceID := rootSpan.SpanContext().TraceID().String()

	emitLogger(t, loggerType, provider, rootCtx, "root")

	httpServerTraceID := ""
	httpServer := httptest.NewServer(otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCtx := r.Context()
		httpServerTraceID = trace.SpanContextFromContext(serverCtx).TraceID().String()
		emitLogger(t, loggerType, provider, serverCtx, "http-server")
		requestCounter.Add(serverCtx, 1, metric.WithAttributes(attribute.String("transport", "http")))
		histogram.Record(serverCtx, 12.5, metric.WithAttributes(attribute.String("transport", "http")))
		_, _ = w.Write([]byte("ok"))
	}), "http-server"))
	defer httpServer.Close()

	httpCtx, httpSpan := tracer.Start(rootCtx, "http-client-call")
	httpRequest, err := http.NewRequestWithContext(httpCtx, http.MethodGet, httpServer.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("failed to create http request: %v", err)
	}
	emitLogger(t, loggerType, provider, httpCtx, "http-client")
	httpClient := &http.Client{Transport: NewOtelHttpTransport()}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	_ = httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		t.Fatalf("unexpected http status: %d", httpResponse.StatusCode)
	}
	requestCounter.Add(httpCtx, 1, metric.WithAttributes(attribute.String("transport", "http-client")))
	histogram.Record(httpCtx, 8.25, metric.WithAttributes(attribute.String("transport", "http-client")))
	httpSpan.End()

	grpcServerTraceID := ""
	grpcUnaryHandler := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		grpcServerTraceID = trace.SpanContextFromContext(ctx).TraceID().String()
		emitLogger(t, loggerType, provider, ctx, "grpc-server")
		requestCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc")))
		histogram.Record(ctx, 21.0, metric.WithAttributes(attribute.String("transport", "grpc")))
		activeCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc")))
		defer activeCounter.Add(ctx, -1, metric.WithAttributes(attribute.String("transport", "grpc")))
		return handler(ctx, req)
	}

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer(
		WithOtelGRPCServerOption(),
		grpc.UnaryInterceptor(grpcUnaryHandler),
	)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("integration.Service", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	grpcCtx, grpcSpan := tracer.Start(rootCtx, "grpc-client-call")
	emitLogger(t, loggerType, provider, grpcCtx, "grpc-client")
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		WithOtelGRPCClientOption(),
	)
	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	grpcClient := healthpb.NewHealthClient(conn)
	if _, err := grpcClient.Check(grpcCtx, &healthpb.HealthCheckRequest{Service: "integration.Service"}); err != nil {
		t.Fatalf("grpc health check failed: %v", err)
	}
	requestCounter.Add(grpcCtx, 1, metric.WithAttributes(attribute.String("transport", "grpc-client")))
	histogram.Record(grpcCtx, 34.0, metric.WithAttributes(attribute.String("transport", "grpc-client")))
	grpcSpan.End()

	requestCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
	histogram.Record(rootCtx, 5.0, metric.WithAttributes(attribute.String("transport", "root")))
	activeCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
	activeCounter.Add(rootCtx, -1, metric.WithAttributes(attribute.String("transport", "root")))

	rootSpan.End()

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

	metricsServer := httptest.NewServer(promhttp.Handler())
	defer metricsServer.Close()

	metricsResp, err := http.Get(metricsServer.URL)
	if err != nil {
		t.Fatalf("failed to scrape prometheus metrics: %v", err)
	}
	defer metricsResp.Body.Close()

	// metricsBody, err := io.ReadAll(metricsResp.Body)
	// if err != nil {
	// 	t.Fatalf("failed to read metrics body: %v", err)
	// }

	// metricsText := string(metricsBody)
	// if !strings.Contains(metricsText, metricPrefix+"request_counter") {
	// 	t.Fatalf("expected prometheus output to include %q, got body: %s", metricPrefix+"request_counter", metricsText)
	// }
	// if !strings.Contains(metricsText, metricPrefix+"latency_ms") {
	// 	t.Fatalf("expected prometheus output to include %q, got body: %s", metricPrefix+"latency_ms", metricsText)
	// }

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown provider: %v", err)
	}
	globalProvider = nil
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
	case LoggerTypeZap:
		zap.L().Debug(message+" debug", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Info(message+" info", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Warn(message+" warn", zap.Any("context", ctx), zap.String("phase", phase))
		zap.L().Error(message+" error", zap.Any("context", ctx), zap.String("phase", phase))
	case LoggerTypeLogrus:
		logrus.WithContext(ctx).WithField("phase", phase).Debug(message + " debug")
		logrus.WithContext(ctx).WithField("phase", phase).Info(message + " info")
		logrus.WithContext(ctx).WithField("phase", phase).Warn(message + " warn")
		logrus.WithContext(ctx).WithField("phase", phase).Error(message + " error")
	case LoggerTypeLogr:
		logger := logr.New(otellogr.NewLogSink("integration", otellogr.WithLoggerProvider(provider.GetLoggerProvider())))
		logger.WithValues("context", ctx, "phase", phase).V(1).Info(message + " debug")
		logger.WithValues("context", ctx, "phase", phase).Info(message + " info")
		logger.WithValues("context", ctx, "phase", phase).Error(fmt.Errorf("synthetic error"), message+" error")
	default:
		t.Fatalf("unsupported logger type: %s", loggerType)
	}
}
