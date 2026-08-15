package test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	grpcpkg "github.com/LinPr/go-bootstrap/grpc"
	httppkg "github.com/LinPr/go-bootstrap/http"
	"github.com/LinPr/go-bootstrap/otel"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelgo "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestOtelHttpGrpcIntegration(t *testing.T) {
	previousSlog := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previousSlog)
	})

	config := &otel.Config{
		ServiceName:    "go-bootstrap-integration",
		ServiceVersion: "1.0.0",
		Log: otel.LogConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/logs",
			Level:      "info",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-integration",
			},
			Logger: otel.LoggerTypeSlog,
			Pretty: false,
		},
		Trace: otel.TraceConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/traces",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-integration",
			},
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: otel.MetricConfig{
			Enable:     true,
			Exporter:   otel.ExporterTypeHTTP,
			RemoteAddr: "http://10.86.11.34:5318/v1/metrics",
			Headers: map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDb21wbGV4cGFzcyMxMjM=",
				"stream-name":   "go-bootstrap-integration",
			},
			Pretty:          false,
			IntervalSeconds: 1,
		},
	}

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

	meter := otelgo.Meter("go-bootstrap-integration")
	metricPrefix := "go_bootstrap_integration_"
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

	tracer := otelgo.Tracer("integration-test-tracer")

	var httpServerTraceID, grpcServerTraceID string
	httpServer := httptest.NewServer(
		httppkg.NewHttpServerOtelHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverCtx := r.Context()
			httpServerTraceID = trace.SpanContextFromContext(serverCtx).TraceID().String()
			slog.InfoContext(serverCtx, "HTTP server received request", "path", r.URL.Path, "method", r.Method)
			requestCounter.Add(serverCtx, 1, metric.WithAttributes(attribute.String("transport", "http-server")))
			histogram.Record(serverCtx, 12.5, metric.WithAttributes(attribute.String("transport", "http-server")))
			_, _ = w.Write([]byte("ok"))
		}), "http-server"))
	defer httpServer.Close()

	httpClient := &http.Client{Transport: httppkg.NewHttpClientOtelTransport()}

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
		grpcpkg.WithOtelGRPCServerStatsHandler(),
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
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcpkg.WithOtelGRPCClientStatsHandler(),
	)

	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	grpcClient := healthpb.NewHealthClient(conn)

	const iterations = 5
	for i := 0; i < iterations; i++ {
		rootCtx, rootSpan := tracer.Start(context.Background(), fmt.Sprintf("integration-root-%d", i))
		rootTraceID := rootSpan.SpanContext().TraceID().String()

		slog.InfoContext(rootCtx, "Starting integration test iteration", "iteration", i, "trace_id", rootTraceID)

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
		slog.InfoContext(grpcCtx, "Sending gRPC health check", "iteration", i, "service", "integration.Service")
		resp, err := grpcClient.Check(grpcCtx, &healthpb.HealthCheckRequest{Service: "integration.Service"})
		if err != nil {
			t.Fatalf("iteration %d: grpc health check failed: %v", i, err)
		}
		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("iteration %d: unexpected grpc status: %v", i, resp.Status)
		}
		requestCounter.Add(grpcCtx, 1, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		histogram.Record(grpcCtx, 34.0, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		grpcSpan.End()

		requestCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		histogram.Record(rootCtx, 5.0, metric.WithAttributes(attribute.String("transport", "root")))
		activeCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		activeCounter.Add(rootCtx, -1, metric.WithAttributes(attribute.String("transport", "root")))

		rootSpan.End()
		slog.InfoContext(rootCtx, "Completed integration test iteration", "iteration", i)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown provider: %v", err)
	}
}
