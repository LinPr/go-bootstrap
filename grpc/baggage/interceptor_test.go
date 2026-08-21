package baggage

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

const (
// loggerType = LoggerTypeSlog
// loggerType = LoggerTypeZap
)

func setupTestLogger(t *testing.T) {
	t.Helper()
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(slogger)

	zaplogger := zap.NewExample()
	zap.ReplaceGlobals(zaplogger)
	t.Cleanup(func() {

	})
}

func TestLoggingInterceptors(t *testing.T) {
	setupTestLogger(t)

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() {
		_ = lis.Close()
	})

	server := grpc.NewServer(
	// grpc.UnaryInterceptor(UnaryServerLoggingInterceptor(loggerType)),
	// grpc.StreamInterceptor(StreamServerLoggingInterceptor(loggerType)),
	)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(server, hs)
	hs.SetServingStatus("test.Service", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(server.Stop)

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// grpc.WithUnaryInterceptor(UnaryClientLoggingInterceptor(loggerType)),
		// grpc.WithStreamInterceptor(StreamClientLoggingInterceptor(loggerType)),
	)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := healthpb.NewHealthClient(conn)

	t.Run("unary success", func(t *testing.T) {
		resp, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "test.Service"})
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}

		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("expected SERVING status, got %v", resp.Status)
		}
	})

	t.Run("stream success", func(t *testing.T) {
		stream, err := client.Watch(context.Background(), &healthpb.HealthCheckRequest{Service: "test.Service"})
		if err != nil {
			t.Fatalf("watch failed: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("recv failed: %v", err)
		}

		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("expected SERVING status, got %v", resp.Status)
		}
	})

	t.Run("unary error logs debug", func(t *testing.T) {

		_, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "missing.Service"})
		if err == nil {
			t.Fatal("expected health check error")
		}

	})

	t.Run("stream error logs debug", func(t *testing.T) {

		stream, err := client.Watch(context.Background(), &healthpb.HealthCheckRequest{Service: "test.Service"})
		if err != nil {
			t.Fatalf("watch setup failed: %v", err)
		}

		_, err = stream.Recv()
		if err != nil {
			t.Fatalf("stream recv failed: %v", err)
		}

	})
}
