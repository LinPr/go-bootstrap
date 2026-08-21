package grpc_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	bsgrpc "github.com/LinPr/go-bootstrap/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func setupTestLogger(t *testing.T) {
	t.Helper()
	previousLogger := slog.Default()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
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
		grpc.UnaryInterceptor(bsgrpc.UnaryServerLoggingInterceptor()),
		grpc.StreamInterceptor(bsgrpc.StreamServerLoggingInterceptor()),
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
		grpc.WithUnaryInterceptor(bsgrpc.UnaryClientLoggingInterceptor()),
		grpc.WithStreamInterceptor(bsgrpc.StreamClientLoggingInterceptor()),
	)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := healthpb.NewHealthClient(conn)

	t.Run("unary", func(t *testing.T) {
		resp, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "test.Service"})
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}

		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Fatalf("expected SERVING status, got %v", resp.Status)
		}
	})

	t.Run("stream", func(t *testing.T) {
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

}
