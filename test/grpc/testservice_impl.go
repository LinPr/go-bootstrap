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

	slog.InfoContext(ctx, "Echo method called")

	return &api.EchoResponse{
		Message:      fmt.Sprintf("Echo: %s", req.Message),
		BaggageValue: baggageValue,
	}, nil
}

func (s *testServiceServer) StreamEcho(req *api.StreamRequest, stream api.TestService_StreamEchoServer) error {
	ctx := stream.Context()
	bag := baggage.FromContext(ctx)
	baggageValue := bag.Member("BaggageKey").Value()

	slog.InfoContext(ctx, "StreamEcho method called")

	for i := int32(0); i < req.Count; i++ {
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
