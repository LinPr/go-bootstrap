package transport

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func NewOtelHttpTransport() *otelhttp.Transport {
	return otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
		}),
	)
}

// WithOtelGRPCClientOption returns a gRPC dial option that enables OTEL
// client-side instrumentation.
func WithOtelGRPCClientOption(opts ...otelgrpc.Option) grpc.DialOption {
	opts = append(
		[]otelgrpc.Option{
			otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
		},
		// Add any user-provided options after the default options so that they can override them.
		opts...,
	)
	return grpc.WithStatsHandler(&chainedStatsHandler{
		handlers: []stats.Handler{
			otelgrpc.NewClientHandler(opts...),
			newClientMessageSizeStatsHandler(),
		},
	})
}

// WithOtelGRPCServerOption returns a gRPC server option that enables OTEL
// server-side instrumentation.
func WithOtelGRPCServerOption(opts ...otelgrpc.Option) grpc.ServerOption {
	opts = append(
		[]otelgrpc.Option{
			otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
		},
		// Add any user-provided options after the default options so that they can override them.
		opts...,
	)
	return grpc.StatsHandler(&chainedStatsHandler{
		handlers: []stats.Handler{
			otelgrpc.NewServerHandler(opts...),
			newMessageSizeStatsHandler(),
		},
	})
}

// chainedStatsHandler chains multiple stats handlers together.
type chainedStatsHandler struct {
	handlers []stats.Handler
}

func (h *chainedStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	for _, handler := range h.handlers {
		ctx = handler.TagRPC(ctx, info)
	}
	return ctx
}

func (h *chainedStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	for _, handler := range h.handlers {
		handler.HandleRPC(ctx, s)
	}
}

func (h *chainedStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	for _, handler := range h.handlers {
		ctx = handler.TagConn(ctx, info)
	}
	return ctx
}

func (h *chainedStatsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
	for _, handler := range h.handlers {
		handler.HandleConn(ctx, s)
	}
}

// messageSizeStatsHandler records gRPC message sizes as metrics.
type messageSizeStatsHandler struct {
	requestSize  metric.Int64Histogram
	responseSize metric.Int64Histogram
}

// newMessageSizeStatsHandler creates a stats handler that records message sizes for server.
func newMessageSizeStatsHandler() *messageSizeStatsHandler {
	meter := otel.Meter("github.com/LinPr/go-bootstrap/otel/transport")
	requestSize, _ := meter.Int64Histogram(
		"rpc.server.request.size",
		metric.WithDescription("Size of RPC request messages in bytes"),
		metric.WithUnit("By"),
	)
	responseSize, _ := meter.Int64Histogram(
		"rpc.server.response.size",
		metric.WithDescription("Size of RPC response messages in bytes"),
		metric.WithUnit("By"),
	)
	return &messageSizeStatsHandler{
		requestSize:  requestSize,
		responseSize: responseSize,
	}
}

// newClientMessageSizeStatsHandler creates a stats handler that records message sizes for client.
func newClientMessageSizeStatsHandler() *messageSizeStatsHandler {
	meter := otel.Meter("github.com/LinPr/go-bootstrap/otel/transport")
	requestSize, _ := meter.Int64Histogram(
		"rpc.client.request.size",
		metric.WithDescription("Size of RPC request messages in bytes"),
		metric.WithUnit("By"),
	)
	responseSize, _ := meter.Int64Histogram(
		"rpc.client.response.size",
		metric.WithDescription("Size of RPC response messages in bytes"),
		metric.WithUnit("By"),
	)
	return &messageSizeStatsHandler{
		requestSize:  requestSize,
		responseSize: responseSize,
	}
}

func (h *messageSizeStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *messageSizeStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch v := s.(type) {
	case *stats.InPayload:
		h.requestSize.Record(ctx, int64(v.Length))
	case *stats.OutPayload:
		h.responseSize.Record(ctx, int64(v.Length))
	}
}

func (h *messageSizeStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *messageSizeStatsHandler) HandleConn(context.Context, stats.ConnStats) {}

// WithMessageSizeMetrics returns a gRPC server option that records message size metrics.
func WithMessageSizeMetrics() grpc.ServerOption {
	return grpc.StatsHandler(newMessageSizeStatsHandler())
}

