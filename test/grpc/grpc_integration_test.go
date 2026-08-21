package test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	bsgrpc "github.com/LinPr/go-bootstrap/grpc"
	"github.com/LinPr/go-bootstrap/otel"
	"github.com/LinPr/go-bootstrap/test/config"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otelgo "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// startHealthServer starts a bufconn gRPC server with the health service registered.
// The serverTraceID pointer is written on each unary call for trace propagation assertions.
func startHealthServer(t *testing.T, serverTraceID *string, meters integrationMeters) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)

	traceCapture := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		*serverTraceID = trace.SpanContextFromContext(ctx).TraceID().String()
		slog.InfoContext(ctx, "gRPC server received request", "method", info.FullMethod)
		meters.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		meters.latency.Record(ctx, 21.0, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		meters.active.Add(ctx, 1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		defer meters.active.Add(ctx, -1, metric.WithAttributes(attribute.String("transport", "grpc-server")))
		return handler(ctx, req)
	}

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.StatsHandler(bsgrpc.NewServerMessageSizeStatsHandler()),
		grpc.ChainUnaryInterceptor(
			traceCapture,
			bsgrpc.UnaryServerLoggingInterceptor(),
			bsgrpc.UnaryServerBaggageInterceptor("BaggageKey"),
		),
		grpc.ChainStreamInterceptor(
			bsgrpc.StreamServerLoggingInterceptor(),
			bsgrpc.StreamServerBaggageInterceptor("BaggageKey"),
		),
	)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("integration.Service", healthpb.HealthCheckResponse_SERVING)

	t.Cleanup(func() { srv.GracefulStop() })
	go func() {
		_ = srv.Serve(lis)
	}()
	return lis
}

// newHealthClient creates a gRPC health client connected via bufconn.
func newHealthClient(t *testing.T, lis *bufconn.Listener) healthpb.HealthClient {
	t.Helper()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithStatsHandler(bsgrpc.NewClientMessageSizeStatsHandler()),
	)
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return healthpb.NewHealthClient(conn)
}

// integrationMeters groups the OTel instruments used by TestGrpcOtelIntegration.
type integrationMeters struct {
	requests metric.Int64Counter
	latency  metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func newIntegrationMeters(t *testing.T) integrationMeters {
	t.Helper()
	meter := otelgo.Meter("go-bootstrap-grpc-integration")
	prefix := "go_bootstrap_grpc_integration_"

	requests, err := meter.Int64Counter(prefix + "request_counter")
	if err != nil {
		t.Fatalf("failed to create request counter: %v", err)
	}
	latency, err := meter.Float64Histogram(prefix + "latency_ms")
	if err != nil {
		t.Fatalf("failed to create latency histogram: %v", err)
	}
	active, err := meter.Int64UpDownCounter(prefix + "active_requests")
	if err != nil {
		t.Fatalf("failed to create active counter: %v", err)
	}
	return integrationMeters{requests: requests, latency: latency, active: active}
}

func TestGrpcOtelIntegration(t *testing.T) {
	_ = config.SetupTestOtelProvider(t)

	meters := newIntegrationMeters(t)
	tracer := otelgo.Tracer("grpc-integration-test-tracer")

	var serverTraceID string
	lis := startHealthServer(t, &serverTraceID, meters)
	grpcClient := newHealthClient(t, lis)

	const iterations = 5
	for i := 0; i < iterations; i++ {
		rootCtx, rootSpan := tracer.Start(context.Background(), fmt.Sprintf("grpc-integration-root-%d", i))
		rootTraceID := rootSpan.SpanContext().TraceID().String()
		slog.InfoContext(rootCtx, "Starting gRPC integration test iteration", "iteration", i, "trace_id", rootTraceID)

		grpcCtx, grpcSpan := tracer.Start(rootCtx, "grpc-client-call")
		slog.InfoContext(grpcCtx, "Sending gRPC health check", "iteration", i, "service", "integration.Service")
		grpcCtx = metadata.NewOutgoingContext(grpcCtx, metadata.Pairs("BaggageKey", "test-baggage-value"))

		resp, err := grpcClient.Check(grpcCtx, &healthpb.HealthCheckRequest{Service: "integration.Service"})
		if err != nil {
			t.Fatalf("iteration %d: grpc health check failed: %v", i, err)
		}
		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("iteration %d: unexpected grpc status: %v", i, resp.Status)
		}
		meters.requests.Add(grpcCtx, 1, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		meters.latency.Record(grpcCtx, 34.0, metric.WithAttributes(attribute.String("transport", "grpc-client")))
		grpcSpan.End()

		meters.requests.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		meters.latency.Record(rootCtx, 5.0, metric.WithAttributes(attribute.String("transport", "root")))
		meters.active.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		meters.active.Add(rootCtx, -1, metric.WithAttributes(attribute.String("transport", "root")))
		rootSpan.End()
		slog.InfoContext(rootCtx, "Completed gRPC integration test iteration", "iteration", i)

		if i == 0 {
			if serverTraceID == "" {
				t.Fatal("grpc server trace id is empty")
			}
			if serverTraceID != rootTraceID {
				t.Fatalf("expected grpc trace id %s, got %s", rootTraceID, serverTraceID)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Logf("warning: failed to shutdown provider: %v", err)
	}
}
