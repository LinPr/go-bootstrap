package transport

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
			slog.DebugContext(ctx, "grpc client stream error", "method", method, "error", err)
		} else {
			slog.DebugContext(ctx, "grpc client stream", "method", method)
		}
		return stream, err
	}
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
		err := handler(srv, ss)

		if err != nil {
			slog.DebugContext(ctx, "grpc server stream error", "method", info.FullMethod, "error", err)
		} else {
			slog.DebugContext(ctx, "grpc server stream", "method", info.FullMethod)
		}
		return err
	}
}
