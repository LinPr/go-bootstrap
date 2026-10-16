package transform

import (
	"encoding/json"
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

func (r RecordJSON) MarshalJSON() ([]byte, error) {
	type Alias RecordJSON
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(&r),
	}
	data, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	for k, v := range r.Attributes {
		m[k] = v
	}

	return json.Marshal(m)
}
