package transport

import (
	"context"

	"github.com/LinPr/go-bootstrap/otel/version"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// GRPCClientStatsOption configures gRPC client stats handlers.
type GRPCClientStatsOption func(*grpcClientStatsConfig)

type grpcClientStatsConfig struct {
	otelOptions    []otelgrpc.Option
	customHandlers []stats.Handler
}

// WithOtelGRPCOptions adds OpenTelemetry options for gRPC client instrumentation.
func WithOtelGRPCOptions(opts ...otelgrpc.Option) GRPCClientStatsOption {
	return func(c *grpcClientStatsConfig) {
		c.otelOptions = append(c.otelOptions, opts...)
	}
}

// WithCustomStatsHandlers adds custom stats handlers for gRPC client.
func WithCustomStatsHandlers(handlers ...stats.Handler) GRPCClientStatsOption {
	return func(c *grpcClientStatsConfig) {
		c.customHandlers = append(c.customHandlers, handlers...)
	}
}

// WithOtelGRPCClientStatsHandler returns a gRPC dial option that enables OTEL
// client-side instrumentation with optional custom stats handlers.
func WithOtelGRPCClientStatsHandler(opts ...GRPCClientStatsOption) grpc.DialOption {
	config := &grpcClientStatsConfig{}
	for _, opt := range opts {
		opt(config)
	}

	otelOpts := append(
		[]otelgrpc.Option{
			otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
		},
		config.otelOptions...,
	)

	handlers := []stats.Handler{
		otelgrpc.NewClientHandler(otelOpts...),
		newClientMessageSizeStatsHandler(),
	}
	handlers = append(handlers, config.customHandlers...)

	return grpc.WithStatsHandler(&chainedStatsHandler{
		handlers: handlers,
	})
}

// GRPCServerStatsOption configures gRPC server stats handlers.
type GRPCServerStatsOption func(*grpcServerStatsConfig)

type grpcServerStatsConfig struct {
	otelOptions    []otelgrpc.Option
	customHandlers []stats.Handler
}

// WithOtelGRPCServerOptions adds OpenTelemetry options for gRPC server instrumentation.
func WithOtelGRPCServerOptions(opts ...otelgrpc.Option) GRPCServerStatsOption {
	return func(c *grpcServerStatsConfig) {
		c.otelOptions = append(c.otelOptions, opts...)
	}
}

// WithCustomServerStatsHandlers adds custom stats handlers for gRPC server.
func WithCustomServerStatsHandlers(handlers ...stats.Handler) GRPCServerStatsOption {
	return func(c *grpcServerStatsConfig) {
		c.customHandlers = append(c.customHandlers, handlers...)
	}
}

// WithOtelGRPCServerStatsHandler returns a gRPC server option that enables OTEL
// server-side instrumentation with optional custom stats handlers.
func WithOtelGRPCServerStatsHandler(opts ...GRPCServerStatsOption) grpc.ServerOption {
	config := &grpcServerStatsConfig{}
	for _, opt := range opts {
		opt(config)
	}

	otelOpts := append(
		[]otelgrpc.Option{
			otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
		},
		config.otelOptions...,
	)

	handlers := []stats.Handler{
		otelgrpc.NewServerHandler(otelOpts...),
		newMessageSizeStatsHandler(),
	}
	handlers = append(handlers, config.customHandlers...)

	return grpc.StatsHandler(&chainedStatsHandler{
		handlers: handlers,
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

// serverMessageSizeStatsHandler records gRPC message sizes for server.
type serverMessageSizeStatsHandler struct {
	inPayloadSize  metric.Int64Histogram
	outPayloadSize metric.Int64Histogram
}

// newMessageSizeStatsHandler creates a stats handler that records message sizes for server.
func newMessageSizeStatsHandler() *serverMessageSizeStatsHandler {
	meter := otel.Meter(
		"github.com/LinPr/go-bootstrap/otel/transport",
		metric.WithInstrumentationVersion(version.Version),
	)
	inPayloadSize, _ := meter.Int64Histogram(
		"rpc.server.inpayload.size",
		metric.WithDescription("Size of RPC incoming payload in bytes"),
		metric.WithUnit("By"),
	)
	outPayloadSize, _ := meter.Int64Histogram(
		"rpc.server.outpayload.size",
		metric.WithDescription("Size of RPC outgoing payload in bytes"),
		metric.WithUnit("By"),
	)
	return &serverMessageSizeStatsHandler{
		inPayloadSize:  inPayloadSize,
		outPayloadSize: outPayloadSize,
	}
}

func (h *serverMessageSizeStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *serverMessageSizeStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch v := s.(type) {
	case *stats.InPayload:
		if v.Length > 0 {
			h.inPayloadSize.Record(ctx, int64(v.Length))
		}
	case *stats.OutPayload:
		if v.Length > 0 {
			h.outPayloadSize.Record(ctx, int64(v.Length))
		}
	}
}

func (h *serverMessageSizeStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *serverMessageSizeStatsHandler) HandleConn(context.Context, stats.ConnStats) {}

// clientMessageSizeStatsHandler records gRPC message sizes for client.
type clientMessageSizeStatsHandler struct {
	inPayloadSize  metric.Int64Histogram
	outPayloadSize metric.Int64Histogram
}

// newClientMessageSizeStatsHandler creates a stats handler that records message sizes for client.
func newClientMessageSizeStatsHandler() *clientMessageSizeStatsHandler {
	meter := otel.Meter(
		"github.com/LinPr/go-bootstrap/otel/transport",
		metric.WithInstrumentationVersion(version.Version),
	)
	inPayloadSize, _ := meter.Int64Histogram(
		"rpc.client.inpayload.size",
		metric.WithDescription("Size of RPC incoming payload in bytes"),
		metric.WithUnit("By"),
	)
	outPayloadSize, _ := meter.Int64Histogram(
		"rpc.client.outpayload.size",
		metric.WithDescription("Size of RPC outgoing payload in bytes"),
		metric.WithUnit("By"),
	)
	return &clientMessageSizeStatsHandler{
		inPayloadSize:  inPayloadSize,
		outPayloadSize: outPayloadSize,
	}
}

func (h *clientMessageSizeStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *clientMessageSizeStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch v := s.(type) {
	case *stats.InPayload:
		if v.Length > 0 {
			h.inPayloadSize.Record(ctx, int64(v.Length))
		}
	case *stats.OutPayload:
		if v.Length > 0 {
			h.outPayloadSize.Record(ctx, int64(v.Length))
		}
	}
}

func (h *clientMessageSizeStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *clientMessageSizeStatsHandler) HandleConn(context.Context, stats.ConnStats) {}
