package grpc

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/stats"
)

// NewServerMessageSizeStatsHandler creates a stats handler that records message sizes for server.
func NewServerMessageSizeStatsHandler() stats.Handler {
	return newMessageSizeStatsHandler()
}

// NewClientMessageSizeStatsHandler creates a stats handler that records message sizes for client.
func NewClientMessageSizeStatsHandler() stats.Handler {
	return newClientMessageSizeStatsHandler()
}

// serverMessageSizeStatsHandler records gRPC message sizes for server.
type serverMessageSizeStatsHandler struct {
	inPayloadSize  metric.Int64Histogram
	outPayloadSize metric.Int64Histogram
}

// newMessageSizeStatsHandler creates a stats handler that records message sizes for server.
func newMessageSizeStatsHandler() *serverMessageSizeStatsHandler {
	meter := otel.Meter(
		"github.com/LinPr/go-bootstrap/grpc",
		metric.WithInstrumentationVersion(Version),
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
		"github.com/LinPr/go-bootstrap/grpc",
		metric.WithInstrumentationVersion(Version),
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
