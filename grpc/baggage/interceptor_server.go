package baggage

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerBaggageInterceptor extracts OpenTelemetry propagation data from gRPC metadata
// and creates baggage members from specified metadata keys.
func UnaryServerBaggageInterceptor(mdKeys ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		ctx = otel.GetTextMapPropagator().Extract(ctx, MetadataCarrier(md))
		members := baggage.FromContext(ctx).Members()

		for _, key := range mdKeys {
			if values := md.Get(key); len(values) > 0 {
				m, err := baggage.NewMember(key, values[0])
				if err != nil {
					slog.ErrorContext(ctx, "failed to create baggage member", "error", err, "key", key)
					continue
				}
				members = append(members, m)
			}
		}

		newBaggage, err := baggage.New(members...)
		if err != nil {
			slog.ErrorContext(ctx, "failed to create baggage", "error", err)
			return handler(ctx, req)
		}

		ctx = baggage.ContextWithBaggage(ctx, newBaggage)
		return handler(ctx, req)
	}
}

// StreamServerBaggageInterceptor extracts OpenTelemetry propagation data from gRPC metadata
// and creates baggage members from specified metadata keys for streaming RPCs.
func StreamServerBaggageInterceptor(mdKeys ...string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(srv, ss)
		}

		ctx = otel.GetTextMapPropagator().Extract(ctx, MetadataCarrier(md))
		members := baggage.FromContext(ctx).Members()

		for _, key := range mdKeys {
			if values := md.Get(key); len(values) > 0 {
				m, err := baggage.NewMember(key, values[0])
				if err != nil {
					slog.ErrorContext(ctx, "failed to create baggage member", "error", err, "key", key)
					continue
				}
				members = append(members, m)
			}
		}

		newBaggage, err := baggage.New(members...)
		if err != nil {
			slog.ErrorContext(ctx, "failed to create baggage", "error", err)
			return handler(srv, ss)
		}

		ctx = baggage.ContextWithBaggage(ctx, newBaggage)
		wrappedStream := &baggageServerStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrappedStream)
	}
}

type baggageServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *baggageServerStream) Context() context.Context {
	return s.ctx
}
