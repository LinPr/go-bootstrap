package otel

import (
	"context"
	"encoding/json"
	"log/slog"
)

// jsonHandler is a slog.Handler middleware that serializes struct, map,
// and slice attribute values to JSON before delegating to the wrapped handler.
//
// It must run at the slog layer (not as an SDK log.Processor) because the
// otelslog bridge flattens such values via fmt %+v into a plain string. Once
// that happens the original value is gone, so JSON encoding has to occur here
// while the real Go value is still available in the slog.Attr.
type jsonHandler struct {
	slog.Handler
	pretty bool
}

// newjsonHandler wraps h so that composite attribute values are emitted
// as JSON text.
func newjsonHandler(h slog.Handler, pretty bool) slog.Handler {
	return jsonHandler{
		Handler: h,
		pretty:  pretty,
	}
}

// Handle rewrites composite attributes to JSON, then delegates to the wrapped
// handler.
func (h jsonHandler) Handle(ctx context.Context, record slog.Record) error {
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		newRecord.AddAttrs(jsonifyAttr(attr, h.pretty))
		return true
	})
	return h.Handler.Handle(ctx, newRecord)
}

// WithAttrs preserves the middleware around the wrapped handler.
func (h jsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	jsonified := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		jsonified[i] = jsonifyAttr(a, h.pretty)
	}
	return jsonHandler{Handler: h.Handler.WithAttrs(jsonified)}
}

// WithGroup preserves the middleware around the wrapped handler.
func (h jsonHandler) WithGroup(name string) slog.Handler {
	return jsonHandler{Handler: h.Handler.WithGroup(name)}
}

// jsonifyAttr returns attr unchanged unless its value is a struct/map/slice
// (slog.KindAny that is not an error), in which case the value is replaced with
// its JSON encoding.
func jsonifyAttr(attr slog.Attr, pretty bool) slog.Attr {
	v := attr.Value.Resolve()
	if v.Kind() != slog.KindAny {
		return attr
	}
	a := v.Any()
	if _, ok := a.(error); ok {
		return attr
	}
	if pretty {
		b, err := json.MarshalIndent(a, "", "  ")
		if err != nil {
			return attr
		}
		return slog.String(attr.Key, string(b))
	}

	b, err := json.Marshal(a)
	if err != nil {
		return attr
	}
	return slog.String(attr.Key, string(b))
}
