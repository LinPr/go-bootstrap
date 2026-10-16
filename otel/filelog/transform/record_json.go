package transform

import (
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel/attribute"
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

	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		newRecord.Attributes[string(kv.Key)] = kv.Value.AsInterface()
		return true
	})

	return newRecord
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

// func NewRecordJSON(r sdklog.Record) RecordJSON {
// 	res := r.Resource()
// 	newRecord := RecordJSON{
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
// 	newRecord.Timestamp = &timestamp
// 	observedTimestamp := r.ObservedTimestamp()
// 	newRecord.ObservedTimestamp = &observedTimestamp

// 	r.WalkAttributes(func(kv attribute.KeyValue) bool {
// 		newRecord.Attributes = append(newRecord.Attributes, kv)
// 		return true
// 	})

// 	return newRecord
// }
