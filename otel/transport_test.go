package otel

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func ExampleNewOtelHttpTransport() {
	// Build a real HTTP server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Inject OTEL transport into a real client.
	client := &http.Client{
		Transport: NewOtelHttpTransport(),
	}
	resp, err := client.Get(srv.URL + "/ping")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output: 200
}

func ExampleWithOtelGRPCClientOption() {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	// Build a gRPC server with OTEL server option.
	grpcServer := grpc.NewServer(
		WithOtelGRPCServerOption(),
	)
	defer grpcServer.Stop()

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("demo.v1.DemoService", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		WithOtelGRPCClientOption(),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Call the health endpoint as a complete client-side example.
	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "demo.v1.DemoService"})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Status.String())
	// Output: SERVING
}

func ExampleWithOtelGRPCServerOption() {
	// Server-only initialization example.
	server := grpc.NewServer(
		WithOtelGRPCServerOption(),
	)
	defer server.Stop()

	fmt.Println(server != nil)
	// Output: true
}

func TestTransportAndGRPC_EmitTelemetryToOTLPHTTP(t *testing.T) {

	globalProvider = nil

	var receivedRequests int32
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedRequests, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	config := &Config{
		ServiceName:    "transport-e2e",
		ServiceVersion: "1.0.0",
		Log: LogConfig{
			Enable:     true,
			Exporter:   ExporterTypeHTTP,
			RemoteAddr: collector.URL,
			Logger:     LoggerTypeSlog,
			Pretty:     false,
		},
		Trace: TraceConfig{
			Enable:        true,
			Exporter:      ExporterTypeHTTP,
			RemoteAddr:    collector.URL,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:               true,
			Exporter:             ExporterTypeHTTP,
			RemoteAddr:           collector.URL,
			IntervalSeconds:      1,
			EnableRuntimeMetrics: true,
		},
	}

	if _, err := NewOtelProviders(config); err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	// 1) Trigger log provider
	slog.Info("transport e2e log", "case", "transport-and-grpc")

	// 2) Trigger HTTP transport instrumentation
	httpTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer httpTarget.Close()

	httpClient := &http.Client{Transport: NewOtelHttpTransport()}
	httpResp, err := httpClient.Get(httpTarget.URL + "/ping")
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	_ = httpResp.Body.Close()

	// 3) Trigger gRPC client/server instrumentation
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer(WithOtelGRPCServerOption())
	defer grpcServer.Stop()

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("demo.v1.DemoService", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
	if _, err := grpcClient.Check(ctx, &healthpb.HealthCheckRequest{Service: "demo.v1.DemoService"}); err != nil {
		t.Fatalf("grpc health check failed: %v", err)
	}

	// Wait for periodic metric export and flush all providers.
	time.Sleep(2 * time.Second)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	globalProvider = nil

	if atomic.LoadInt32(&receivedRequests) == 0 {
		t.Fatal("expected the local collector to receive at least one OTLP request")
	}

	t.Log("telemetry emitted to remote OTLP HTTP collector; verify logs/traces/metrics in backend")
}
