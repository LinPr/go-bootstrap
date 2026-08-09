package otel

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
)

func NewOtelHttpTransport() *otelhttp.Transport {
	return otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
		}),
	)
}

// NewOtelGRPCClientDialOption returns a gRPC dial option that enables OTEL
// client-side instrumentation.
func NewOtelGRPCClientDialOption(opts ...otelgrpc.Option) grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler(opts...))
}

// NewOtelGRPCServerOption returns a gRPC server option that enables OTEL
// server-side instrumentation.
func NewOtelGRPCServerOption(opts ...otelgrpc.Option) grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler(opts...))
}
