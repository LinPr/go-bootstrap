package grpc

import (
	"context"
	"log/slog"
	"maps"
	"slices"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type MetadataCarrier metadata.MD

// assert that MetadataCarrier implements the TextMapCarrier interface.
var _ propagation.TextMapCarrier = MetadataCarrier{}

func (m MetadataCarrier) Get(key string) string {
	values := metadata.MD(m).Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (m MetadataCarrier) Set(key, value string) {
	metadata.MD(m).Set(key, value)
}

func (m MetadataCarrier) Keys() []string {
	return slices.Collect(maps.Keys(m))
}

func Inject(ctx context.Context, propagators propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	propagators.Inject(ctx, MetadataCarrier(md))

	return metadata.NewOutgoingContext(ctx, md)
}

func Extract(ctx context.Context, propagators propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	return propagators.Extract(ctx, MetadataCarrier(md))
}

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
