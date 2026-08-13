package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
)

// UnaryClientLoggingInterceptor logs debug information for unary client RPCs.
func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			slog.DebugContext(ctx, "grpc client request error", "method", method, "request", req, "error", err)
		} else {
			slog.DebugContext(ctx, "grpc client", "method", method, "request", req, "response", reply)
		}
		return err
	}
}

// StreamClientLoggingInterceptor logs debug information for streaming client RPCs.
func StreamClientLoggingInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			slog.DebugContext(ctx, "grpc client stream creation error", "method", method, "error", err)
			return nil, err
		}

		slog.DebugContext(ctx, "grpc client stream created", "method", method)
		return &loggingClientStream{ClientStream: stream, ctx: ctx, method: method}, nil
	}
}

type loggingClientStream struct {
	grpc.ClientStream
	ctx    context.Context
	method string
}

func (s *loggingClientStream) RecvMsg(m any) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		slog.DebugContext(s.ctx, "grpc client stream recv error", "method", s.method, "error", err)
	} else {
		slog.DebugContext(s.ctx, "grpc client stream recv", "method", s.method, "message", m)
	}
	return err
}

func (s *loggingClientStream) SendMsg(m any) error {
	err := s.ClientStream.SendMsg(m)
	if err != nil {
		slog.DebugContext(s.ctx, "grpc client stream send error", "method", s.method, "error", err)
	} else {
		slog.DebugContext(s.ctx, "grpc client stream send", "method", s.method, "message", m)
	}
	return err
}

// UnaryServerLoggingInterceptor logs debug information for unary server RPCs.
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			slog.DebugContext(ctx, "grpc server request error", "method", info.FullMethod, "request", req, "error", err)
		} else {
			slog.DebugContext(ctx, "grpc server", "method", info.FullMethod, "request", req, "response", resp)
		}
		return resp, err
	}
}

// StreamServerLoggingInterceptor logs debug information for streaming server RPCs.
func StreamServerLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		wrappedStream := &loggingServerStream{ServerStream: ss, ctx: ctx, method: info.FullMethod}
		err := handler(srv, wrappedStream)

		if err != nil {
			slog.DebugContext(ctx, "grpc server stream ended with error", "method", info.FullMethod, "error", err)
		} else {
			slog.DebugContext(ctx, "grpc server stream ended", "method", info.FullMethod)
		}
		return err
	}
}

type loggingServerStream struct {
	grpc.ServerStream
	ctx    context.Context
	method string
}

func (s *loggingServerStream) Context() context.Context {
	return s.ctx
}

func (s *loggingServerStream) RecvMsg(m any) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		slog.DebugContext(s.ctx, "grpc server stream recv error", "method", s.method, "error", err)
	} else {
		slog.DebugContext(s.ctx, "grpc server stream recv", "method", s.method, "message", m)
	}
	return err
}

func (s *loggingServerStream) SendMsg(m any) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		slog.DebugContext(s.ctx, "grpc server stream send error", "method", s.method, "error", err)
	} else {
		slog.DebugContext(s.ctx, "grpc server stream send", "method", s.method, "message", m)
	}
	return err
}
