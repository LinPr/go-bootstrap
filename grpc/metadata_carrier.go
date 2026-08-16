package grpc

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
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
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
