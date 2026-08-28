package test

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LinPr/go-bootstrap/test/grpc/api"
	"go.opentelemetry.io/otel/baggage"
)

type testServiceServer struct {
	api.UnimplementedTestServiceServer
}

func (s *testServiceServer) Echo(ctx context.Context, req *api.EchoRequest) (*api.EchoResponse, error) {
	bag := baggage.FromContext(ctx)
	baggageValue := bag.Member("BaggageKey").Value()

	slog.InfoContext(ctx, "Unary Echo method called, baggage kv"+fmt.Sprintf("BaggageKey=%s", baggageValue))

	return &api.EchoResponse{
		Message:      fmt.Sprintf("Echo: %s", req.Message),
		BaggageValue: baggageValue,
	}, nil
}

func (s *testServiceServer) StreamEcho(req *api.StreamRequest, stream api.TestService_StreamEchoServer) error {
	ctx := stream.Context()
	bag := baggage.FromContext(ctx)
	baggageValue := bag.Member("BaggageKey").Value()

	slog.InfoContext(ctx, "Stream Echo method called, baggage kv"+fmt.Sprintf("BaggageKey=%s", baggageValue))

	for i := range req.Count {
		resp := &api.StreamResponse{
			Message:      fmt.Sprintf("Stream %s", req.Message),
			Index:        i,
			BaggageValue: baggageValue,
		}

		slog.InfoContext(ctx, "Sending stream response")

		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	return nil
}
