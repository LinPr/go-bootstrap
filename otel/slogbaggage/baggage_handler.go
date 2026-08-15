package slogbaggage

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/baggage"
)

type baggageHandler struct {
	handler        slog.Handler
	baggageMembers map[string]struct{}
}

func (bh *baggageHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return bh.handler.Enabled(ctx, level)
}

func (bh *baggageHandler) Handle(ctx context.Context, record slog.Record) error {
	// Extract baggage members and add them to the record's attributes.
	members := baggage.FromContext(ctx).Members()

	for _, member := range members {
		if _, ok := bh.baggageMembers[member.Key()]; ok {
			record.AddAttrs(slog.String(member.Key(), member.Value()))
		}
	}
	return bh.handler.Handle(ctx, record)
}

func (bh *baggageHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &baggageHandler{
		handler:        bh.handler.WithAttrs(attrs),
		baggageMembers: bh.baggageMembers,
	}
}

func (bh *baggageHandler) WithGroup(name string) slog.Handler {
	return &baggageHandler{
		handler:        bh.handler.WithGroup(name),
		baggageMembers: bh.baggageMembers,
	}
}

// Option configures a baggage handler.
type Option func(*baggageHandler)

// WithBaggageMembers specifies which baggage member keys should be added to log records.
func WithBaggageMembers(members ...string) Option {
	return func(bh *baggageHandler) {
		for _, member := range members {
			bh.baggageMembers[member] = struct{}{}
		}
	}
}

// NewBaggageHandler creates a slog handler that extracts specified baggage members and adds them as log attributes.
func NewBaggageHandler(handler slog.Handler, opts ...Option) slog.Handler {
	bh := &baggageHandler{
		handler:        handler,
		baggageMembers: make(map[string]struct{}),
	}

	for _, ops := range opts {
		ops(bh)
	}

	return bh

}
