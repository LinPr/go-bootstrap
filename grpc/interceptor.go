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

// UnaryServerLoggingInterceptor logs debug information for unary server RPCs.
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			slog.DebugContext(ctx, fmt.Sprintf("grpc server unary handler: %s error: %s", info.FullMethod, err.Error()), "request", req)
			return resp, err
		}

		slog.DebugContext(ctx, fmt.Sprintf("grpc server unary handler: %s", info.FullMethod), "grpc_request", req, "grpc_response", resp)
		return resp, nil
	}
}

// StreamServerLoggingInterceptor logs debug information for streaming server RPCs.
func StreamServerLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrappedStream := &loggingServerStream{ServerStream: ss, method: info.FullMethod}
		if err := handler(srv, wrappedStream); err != nil {
			return err
		}

		return nil
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

type loggingServerStream struct {
	grpc.ServerStream
	method string
}

func (s *loggingServerStream) Context() context.Context {
	return s.ServerStream.Context()
}

func (s *loggingServerStream) RecvMsg(m any) error {
	defer slog.DebugContext(s.ServerStream.Context(), "grpc server stream recv: "+s.method, "grpc_stream_recv", m)
	return s.ServerStream.RecvMsg(m)
}

func (s *loggingServerStream) SendMsg(m any) error {
	defer slog.DebugContext(s.ServerStream.Context(), "grpc server stream send: "+s.method, "grpc_stream_send", m)
	return s.ServerStream.SendMsg(m)
}
