package transform

import (
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/log"
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

func RecordJSONs(records []log.Record) []RecordJSON {
	if len(records) == 0 {
		return nil
	}
	recordJsons := make([]RecordJSON, 0, len(records))
	for _, r := range records {
		recordJson := RecordJSON{
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
		recordJson.Timestamp = &timestamp

		for _, kv := range r.Resource().Attributes() {
			recordJson.Attributes[string(kv.Key)] = kv.Value.AsInterface()
		}

		scopedAttr := r.InstrumentationScope().Attributes
		for _, kv := range scopedAttr.ToSlice() {
			recordJson.Attributes[string(kv.Key)] = kv.Value.AsInterface()
		}

		r.WalkAttributes(func(kv attribute.KeyValue) bool {
			recordJson.Attributes[string(kv.Key)] = kv.Value.AsInterface()
			return true
		})

		recordJsons = append(recordJsons, recordJson)
	}

	return recordJsons
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
// to the attribute, matching the previous map-merge behaviour. Note that JSON
// objects with duplicate keys are legal but ambiguous; most decoders, including
// encoding/json, keep the last occurrence.
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

// // RecordJSON is a JSON-serializable representation of a Record.
// type RecordJSON struct {
// 	Timestamp         *time.Time `json:",omitempty"`
// 	ObservedTimestamp *time.Time `json:",omitempty"`
// 	EventName         string     `json:",omitempty"`
// 	Severity          log.Severity
// 	SeverityText      string
// 	Body              attribute.Value
// 	Attributes        []attribute.KeyValue
// 	TraceID           trace.TraceID
// 	SpanID            trace.SpanID
// 	TraceFlags        trace.TraceFlags
// 	Resource          *resource.Resource
// 	Scope             instrumentation.Scope
// 	DroppedAttributes int
// }

// func RecordJSON(r log.Record) RecordJSON {
// 	res := r.Resource()
// 	recordJson := RecordJSON{
// 		EventName:         r.EventName(),
// 		Severity:          r.Severity(),
// 		SeverityText:      r.SeverityText(),
// 		Body:              r.Body(),
// 		Attributes:        make([]attribute.KeyValue, 0, r.AttributesLen()),
// 		TraceID:           r.TraceID(),
// 		SpanID:            r.SpanID(),
// 		TraceFlags:        r.TraceFlags(),
// 		Resource:          res,
// 		Scope:             r.InstrumentationScope(),
// 		DroppedAttributes: r.DroppedAttributes(),
// 	}

// 	timestamp := r.Timestamp()
// 	recordJson.Timestamp = &timestamp
// 	observedTimestamp := r.ObservedTimestamp()
// 	recordJson.ObservedTimestamp = &observedTimestamp

// 	r.WalkAttributes(func(kv attribute.KeyValue) bool {
// 		recordJson.Attributes = append(recordJson.Attributes, kv)
// 		return true
// 	})

// 	return recordJson
// }
