package transform

import (
	"encoding/json"
	"fmt"
	"time"

	api "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

type Attributes map[string]any

// RecordJSON is a JSON-serializable representation of a Record.
type RecordJSON struct {
	Timestamp         *time.Time       `json:"timestamp,omitempty"`
	EventName         string           `json:"event_name,omitempty"`
	Scope             string           `json:"scope,omitempty"`
	Severity          string           `json:"severity,omitempty"`
	Body              string           `json:"body"`
	TraceID           trace.TraceID    `json:"trace_id"`
	SpanID            trace.SpanID     `json:"span_id"`
	TraceFlags        trace.TraceFlags `json:"trace_flags"`
	DroppedAttributes int              `json:"dropped_attributes"`
	Attributes        `json:"-"`
}

func NewRecordJSON(r sdklog.Record) RecordJSON {

	newRecord := RecordJSON{
		EventName:         r.EventName(),
		Scope:             r.InstrumentationScope().Name,
		Severity:          r.SeverityText(),
		Body:              r.Body().String(),
		TraceID:           r.TraceID(),
		SpanID:            r.SpanID(),
		TraceFlags:        r.TraceFlags(),
		DroppedAttributes: r.DroppedAttributes(),
		Attributes:        make(Attributes, r.AttributesLen()),
	}

	timestamp := r.Timestamp()
	newRecord.Timestamp = &timestamp

	for _, kv := range r.Resource().Attributes() {
		newRecord.Attributes[string(kv.Key)] = kv.Value.AsInterface()
	}

	scopedAttr := r.InstrumentationScope().Attributes
	for _, kv := range scopedAttr.ToSlice() {
		newRecord.Attributes[string(kv.Key)] = kv.Value.AsInterface()
	}

	r.WalkAttributes(func(kv api.KeyValue) bool {
		newRecord.Attributes[kv.Key] = logValueToAny(kv.Value)
		return true
	})

	return newRecord
}

// logValueToAny converts a log record attribute value into something
// encoding/json can marshal.
//
// Resource and scope attributes are [attribute.Value]s and have AsInterface
// for this, but record attributes are [api.Value]s, which do not, so the
// mapping is spelled out here.
func logValueToAny(v api.Value) any {
	switch v.Kind() {
	case api.KindBool:
		return v.AsBool()
	case api.KindFloat64:
		return v.AsFloat64()
	case api.KindInt64:
		return v.AsInt64()
	case api.KindString:
		return v.AsString()
	case api.KindBytes:
		return v.AsBytes()
	case api.KindSlice:
		values := v.AsSlice()
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = logValueToAny(value)
		}
		return out
	case api.KindMap:
		kvs := v.AsMap()
		out := make(map[string]any, len(kvs))
		for _, kv := range kvs {
			out[kv.Key] = logValueToAny(kv.Value)
		}
		return out
	case api.KindEmpty:
		return nil
	default:
		return v.String()
	}
}

// recordAlias strips RecordJSON's MarshalJSON method so json.Marshal encodes
// the declared fields instead of recursing into MarshalJSON. The `json:"-"` tag
// on the promoted Attributes field still applies, so the alias never emits the
// attribute map as a nested object.
type recordAlias RecordJSON

// MarshalJSON encodes the record with its attributes flattened in alongside the
// declared fields.
//
// It marshals the struct once, drops the closing brace, and splices the
// attribute pairs in before closing the object again. The attributes are
// written last so a key that collides with a declared field (say "body") loses
// to the attribute. Note that JSON objects with duplicate keys are legal but
// ambiguous; most decoders, including encoding/json, keep the last occurrence.
func (r RecordJSON) MarshalJSON() ([]byte, error) {
	head, err := json.Marshal(recordAlias(r))
	if err != nil {
		return nil, err
	}

	// head is always a non-empty object: Body, TraceID, SpanID, TraceFlags and
	// DroppedAttributes have no omitempty, so there is a trailing '}' to drop
	// and at least one pair before it.
	buf := make([]byte, 0, len(head)+len(r.Attributes)*32)
	buf = append(buf, head[:len(head)-1]...)

	for k, v := range r.Attributes {
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal attribute %q: %w", k, err)
		}
		buf = append(buf, ',')
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
	}

	return append(buf, '}'), nil
}

// AppendRecordJSON appends the JSON encoding of r to dst and returns the
// extended slice. Callers that encode many records in a row can reuse one
// buffer across calls instead of letting each record allocate its own.
//
// A record that fails to encode is skipped: dst is returned with whatever was
// appended for earlier records intact, and nothing partial for this one. That
// keeps one malformed attribute value from poisoning a whole batch of logs.
func AppendRecordJSON(dst []byte, r sdklog.Record) []byte {
	b, err := NewRecordJSON(r).MarshalJSON()
	if err != nil {
		return dst
	}
	return append(dst, b...)
}
