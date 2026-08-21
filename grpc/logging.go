package grpc

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

// UnaryClientLoggingInterceptor logs debug information for unary client RPCs.
func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			slog.ErrorContext(ctx,
				fmt.Sprintf("grpc client unary invoke: %s error: %s", method, err.Error()),
				"peer", peerAddr(ctx),
				"request", req,
			)
			return err
		}

		slog.DebugContext(ctx,
			fmt.Sprintf("grpc client unary invoke: %s", method),
			"peer", peerAddr(ctx),
			"grpc_request", req,
			"grpc_response", reply,
		)
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
	ctx := s.ClientStream.Context()
	if err := s.ClientStream.RecvMsg(m); err != nil {
		slog.ErrorContext(ctx,
			fmt.Sprintf("grpc client stream recv: %s, error: %s", s.method, err.Error()),
			"peer", peerAddr(ctx),
			"stream_recv", m,
		)
		return err
	}

	slog.DebugContext(ctx,
		"grpc client stream recv: "+s.method,
		"peer", peerAddr(ctx),
		"stream_recv", m,
	)
	return nil
}

func (s *loggingClientStream) SendMsg(m any) error {
	ctx := s.ClientStream.Context()
	if err := s.ClientStream.SendMsg(m); err != nil {
		slog.ErrorContext(ctx,
			fmt.Sprintf("grpc client stream send: %s, error: %s", s.method, err.Error()),
			"peer", peerAddr(ctx),
			"stream_send", m,
		)
		return err
	}

	slog.DebugContext(ctx,
		"grpc client stream send: "+s.method,
		"peer", peerAddr(ctx),
		"stream_send", m,
	)
	return nil
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
