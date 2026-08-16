package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ginpkg "github.com/LinPr/go-bootstrap/gin"
	grpcpkg "github.com/LinPr/go-bootstrap/grpc"
	"github.com/LinPr/go-bootstrap/test/config"
	"github.com/LinPr/go-bootstrap/test/grpc/api"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/baggage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// localTestServiceServer is a local implementation of api.TestService for this package.
type localTestServiceServer struct {
	api.UnimplementedTestServiceServer
}

func (s *localTestServiceServer) Echo(ctx context.Context, req *api.EchoRequest) (*api.EchoResponse, error) {
	bag := baggage.FromContext(ctx)
	baggageValue := bag.Member("BaggageKey").Value()
	slog.InfoContext(ctx, "gin_grpc Echo called")
	return &api.EchoResponse{
		Message:      fmt.Sprintf("Echo: %s", req.Message),
		BaggageValue: baggageValue,
	}, nil
}

func (s *localTestServiceServer) StreamEcho(req *api.StreamRequest, stream api.TestService_StreamEchoServer) error {
	ctx := stream.Context()
	bag := baggage.FromContext(ctx)
	baggageValue := bag.Member("BaggageKey").Value()
	slog.InfoContext(ctx, "gin_grpc StreamEcho called")
	for i := int32(0); i < req.Count; i++ {
		if err := stream.Send(&api.StreamResponse{
			Message:      fmt.Sprintf("Stream %s", req.Message),
			Index:        i,
			BaggageValue: baggageValue,
		}); err != nil {
			return err
		}
	}
	return nil
}

// startGrpcServer starts an in-process gRPC server using bufconn and returns the listener.
func startGrpcServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.StatsHandler(grpcpkg.NewServerMessageSizeStatsHandler()),
		grpc.ChainUnaryInterceptor(
			grpcpkg.UnaryServerBaggageInterceptor("BaggageKey"),
			grpcpkg.UnaryServerLoggingInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcpkg.StreamServerBaggageInterceptor("BaggageKey"),
			grpcpkg.StreamServerLoggingInterceptor(),
		),
	)
	api.RegisterTestServiceServer(srv, &localTestServiceServer{})
	t.Cleanup(func() { srv.GracefulStop() })
	go func() {
		if err := srv.Serve(lis); err != nil {
			// bufconn closed on cleanup — expected
		}
	}()
	return lis
}

// newGrpcClient creates a gRPC client connected via bufconn.
func newGrpcClient(t *testing.T, lis *bufconn.Listener) api.TestServiceClient {
	t.Helper()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithStatsHandler(grpcpkg.NewClientMessageSizeStatsHandler()),
	)
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return api.NewTestServiceClient(conn)
}

// setupGinRouter creates the Gin engine with OTel and baggage middleware, and registers routes.
func setupGinRouter(grpcClient api.TestServiceClient) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(otelgin.Middleware("gin-grpc-test"))
	r.Use(ginpkg.WithOtelBaggageFromHeader("BaggageKey"))

	r.GET("/echo", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resp, err := grpcClient.Echo(ctx, &api.EchoRequest{Message: "hello"})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       resp.Message,
			"baggage_value": resp.BaggageValue,
		})
	})

	r.GET("/stream", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		stream, err := grpcClient.StreamEcho(ctx, &api.StreamRequest{Message: "hello", Count: 3})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var values []string
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			values = append(values, resp.BaggageValue)
		}
		c.JSON(http.StatusOK, gin.H{"baggage_values": values})
	})

	return r
}

func TestGinGrpcIntegration(t *testing.T) {
	providers := config.SetupTestOtelProvider(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = providers.Shutdown(ctx)
	})

	lis := startGrpcServer(t)
	grpcClient := newGrpcClient(t, lis)
	router := setupGinRouter(grpcClient)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	t.Run("UnaryEchoWithBaggage", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/echo", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("BaggageKey", "test-baggage-value")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("http request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("unexpected status %d: %s", resp.StatusCode, body)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		got := result["baggage_value"]
		if got != "test-baggage-value" {
			t.Errorf("baggage_value = %q, want %q", got, "test-baggage-value")
		}
	})

	t.Run("StreamEchoWithBaggage", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/stream", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("BaggageKey", "stream-baggage-value")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("http request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("unexpected status %d: %s", resp.StatusCode, body)
		}

		var result map[string][]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		values := result["baggage_values"]
		if len(values) != 3 {
			t.Fatalf("expected 3 stream responses, got %d", len(values))
		}
		for i, v := range values {
			if v != "stream-baggage-value" {
				t.Errorf("baggage_values[%d] = %q, want %q", i, v, "stream-baggage-value")
			}
		}
	})

	t.Run("NoBaggageHeader", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/echo", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		// no BaggageKey header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("http request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("unexpected status %d: %s", resp.StatusCode, body)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Without baggage header, BaggageKey should be empty
		if got := result["baggage_value"]; got != "" {
			t.Errorf("expected empty baggage_value without header, got %q", got)
		}
	})

}
