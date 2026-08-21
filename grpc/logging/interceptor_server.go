package logging

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// peerAddr extracts the remote peer address from ctx, or "unknown" if absent.
func peerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return "unknown"
}

// UnaryServerLoggingInterceptor logs debug information for unary server RPCs.
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		resp, err := handler(ctx, req)
		if err != nil {
			slog.ErrorContext(ctx,
				fmt.Sprintf("grpc server unary handler: %s error: %s", info.FullMethod, err.Error()),
				"peer", peerAddr(ctx),
				"grpc_request", req,
			)
			return resp, err
		}

		slog.DebugContext(ctx,
			fmt.Sprintf("grpc server unary handler: %s", info.FullMethod),
			"peer", peerAddr(ctx),
			"grpc_request", req,
			"grpc_response", resp,
		)
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

type loggingServerStream struct {
	grpc.ServerStream
	method string
}

func (s *loggingServerStream) RecvMsg(m any) error {
	ctx := s.ServerStream.Context()
	if err := s.ServerStream.RecvMsg(m); err != nil {
		slog.ErrorContext(ctx,
			fmt.Sprintf("grpc server stream recv: %s, error: %s", s.method, err.Error()),
			"peer", peerAddr(ctx),
			"grpc_stream_recv", m,
		)
		return err
	}

	slog.DebugContext(ctx,
		"grpc server stream recv: "+s.method,
		"peer", peerAddr(ctx),
		"grpc_stream_recv", m,
	)
	return nil
}

func (s *loggingServerStream) SendMsg(m any) error {
	ctx := s.ServerStream.Context()
	if err := s.ServerStream.SendMsg(m); err != nil {
		slog.ErrorContext(ctx,
			fmt.Sprintf("grpc server stream send: %s, error: %s", s.method, err.Error()),
			"peer", peerAddr(ctx),
			"grpc_stream_send", m,
		)
		return err
	}

	slog.DebugContext(ctx,
		"grpc server stream send: "+s.method,
		"peer", peerAddr(ctx),
		"grpc_stream_send", m,
	)
	return nil
}
