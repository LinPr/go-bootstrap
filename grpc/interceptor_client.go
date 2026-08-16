package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
)

// UnaryClientLoggingInterceptor logs debug information for unary client RPCs.
func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			slog.DebugContext(ctx, fmt.Sprintf("grpc client unary invoke: %s error: %s", method, err.Error()), "request", req)
			return err
		}

		slog.DebugContext(ctx, fmt.Sprintf("grpc client unary invoke: %s", method), "grpc_request", req, "grpc_response", reply)
		return nil
	}
}

// StreamClientLoggingInterceptor logs debug information for streaming client RPCs.
func StreamClientLoggingInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return &loggingClientStream{ClientStream: stream, method: method}, nil
	}
}

type loggingClientStream struct {
	grpc.ClientStream
	method string
}

func (s *loggingClientStream) RecvMsg(m any) error {
	defer slog.DebugContext(s.ClientStream.Context(), "grpc client stream recv: "+s.method, "stream_recv", m)
	return s.ClientStream.RecvMsg(m)
}

func (s *loggingClientStream) SendMsg(m any) error {
	defer slog.DebugContext(s.ClientStream.Context(), "grpc client stream send: "+s.method, "stream_send", m)
	return s.ClientStream.SendMsg(m)
}
