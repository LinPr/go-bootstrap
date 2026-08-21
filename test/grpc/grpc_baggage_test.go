package test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	bsgrpcbaggage "github.com/LinPr/go-bootstrap/grpc/baggage"
	bsgrpclogging "github.com/LinPr/go-bootstrap/grpc/logging"
	"github.com/LinPr/go-bootstrap/otel"
	"github.com/LinPr/go-bootstrap/test/config"
	"github.com/LinPr/go-bootstrap/test/grpc/api"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestGrpcBaggageInterceptor(t *testing.T) {
	_ = config.SetupTestOtelProvider(t)

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	testSvc := &testServiceServer{}
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			bsgrpclogging.UnaryServerLoggingInterceptor(),
			bsgrpcbaggage.UnaryServerBaggageInterceptor("BaggageKey"),
		),
		grpc.ChainStreamInterceptor(
			bsgrpclogging.StreamServerLoggingInterceptor(),
			bsgrpcbaggage.StreamServerBaggageInterceptor("BaggageKey"),
		),
	)
	api.RegisterTestServiceServer(grpcServer, testSvc)

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
	)
	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	client := api.NewTestServiceClient(conn)

	t.Run("UnaryCallWithBaggage", func(t *testing.T) {
		ctx := context.Background()
		md := metadata.Pairs("BaggageKey", "unary-test-value")
		ctx = metadata.NewOutgoingContext(ctx, md)

		slog.InfoContext(ctx, "Client: calling Echo with baggage")

		resp, err := client.Echo(ctx, &api.EchoRequest{Message: "hello unary"})
		if err != nil {
			t.Fatalf("Echo call failed: %v", err)
		}

		if resp.BaggageValue != "unary-test-value" {
			t.Errorf("expected baggage value 'unary-test-value', got '%s'", resp.BaggageValue)
		}

		slog.InfoContext(ctx, "Client: received Echo response", "message", resp.Message)
	})

	t.Run("StreamCallWithBaggage", func(t *testing.T) {
		ctx := context.Background()
		md := metadata.Pairs("BaggageKey", "stream-test-value")
		ctx = metadata.NewOutgoingContext(ctx, md)

		slog.InfoContext(ctx, "Client: calling StreamEcho with baggage")

		stream, err := client.StreamEcho(ctx, &api.StreamRequest{Message: "hello stream", Count: 3})
		if err != nil {
			t.Fatalf("StreamEcho call failed: %v", err)
		}

		receivedCount := 0
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("stream receive failed: %v", err)
			}

			if resp.BaggageValue != "stream-test-value" {
				t.Errorf("expected baggage value 'stream-test-value', got '%s'", resp.BaggageValue)
			}

			slog.InfoContext(ctx, "Client: received stream response", "index", resp.Index, "message", resp.Message)
			receivedCount++
		}

		if receivedCount != 3 {
			t.Errorf("expected 3 stream responses, got %d", receivedCount)
		}
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Logf("warning: failed to shutdown provider: %v", err)
	}
}
