package test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	grpcpkg "github.com/LinPr/go-bootstrap/grpc"
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

func TestGrpcOtelIntegration(t *testing.T) {
	_ = config.SetupTestOtelProvider(t)

	meter := otelgo.Meter("go-bootstrap-grpc-integration")
	metricPrefix := "go_bootstrap_grpc_integration_"
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

	tracer := otelgo.Tracer("grpc-integration-test-tracer")

	var grpcServerTraceID string
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
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.StatsHandler(grpcpkg.NewServerMessageSizeStatsHandler()),
		grpc.ChainUnaryInterceptor(
			grpcUnaryHandler,
			grpcpkg.UnaryServerLoggingInterceptor(),
			grpcpkg.UnaryServerBaggageInterceptor("BaggageKey"),
		),
		grpc.ChainStreamInterceptor(
			grpcpkg.StreamServerLoggingInterceptor(),
			grpcpkg.StreamServerBaggageInterceptor("BaggageKey"),
		),
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
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithStatsHandler(grpcpkg.NewClientMessageSizeStatsHandler()),
	)

	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	grpcClient := healthpb.NewHealthClient(conn)

	const iterations = 5
	for i := 0; i < iterations; i++ {
		rootCtx, rootSpan := tracer.Start(context.Background(), fmt.Sprintf("grpc-integration-root-%d", i))
		rootTraceID := rootSpan.SpanContext().TraceID().String()

		slog.InfoContext(rootCtx, "Starting gRPC integration test iteration", "iteration", i, "trace_id", rootTraceID)

		grpcCtx, grpcSpan := tracer.Start(rootCtx, "grpc-client-call")
		slog.InfoContext(grpcCtx, "Sending gRPC health check", "iteration", i, "service", "integration.Service")
		md := metadata.Pairs("BaggageKey", "test-baggage-value")
		grpcCtx = metadata.NewOutgoingContext(grpcCtx, md)
		resp, err := grpcClient.Check(
			grpcCtx, &healthpb.HealthCheckRequest{Service: "integration.Service"},
		)
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
		slog.InfoContext(rootCtx, "Completed gRPC integration test iteration", "iteration", i)

		if i == 0 {
			if grpcServerTraceID == "" {
				t.Fatal("grpc server trace id is empty")
			}
			if grpcServerTraceID != rootTraceID {
				t.Fatalf("expected grpc trace id %s, got %s", rootTraceID, grpcServerTraceID)
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
