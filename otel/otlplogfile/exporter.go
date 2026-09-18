package otlplogfile

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"

	"github.com/LinPr/go-bootstrap/otel/otlplogfile/transform"
	"go.opentelemetry.io/otel/sdk/log"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ log.Exporter = &Exporter{}

// Exporter writes JSON-encoded log records to an [io.Writer] ([os.Stdout] by default).
// Exporter must be created with [New].
type Exporter struct {
	writer  io.Writer
	stopped atomic.Bool
	mu      sync.Mutex
	// timestamps bool
	// inst       *observ.Instrumentation
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)

	return &Exporter{
		writer: cfg.Writer,
	}, nil
	// var err error
	// // e.inst, err = observ.NewInstrumentation(counter.NextExporterID())
	// return e, err
}

var transformResourceLogs = transform.ResourceLogs

// Exporter handles the delivery of log records to external receivers.
// Any of the Exporter's methods may be called concurrently with itself or with other methods. It is the responsibility of the Exporter to manage this concurrency.
func (e *Exporter) Export(ctx context.Context, records []log.Record) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	resourceLogs := transformResourceLogs(records)
	if resourceLogs == nil {
		return nil
	}

	pbRequest := &collogpb.ExportLogsServiceRequest{ResourceLogs: resourceLogs}

	b, err := protojson.MarshalOptions{
		UseEnumNumbers: true,
	}.Marshal(pbRequest)
	if err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	decodeSpanIDs(data)

	{
		e.mu.Lock()
		defer e.mu.Unlock()

		if err := json.NewEncoder(e.writer).Encode(&data); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown shuts down the Exporter. Calls to Export after Shutdown return
// [log.ErrExporterShutdown].
func (e *Exporter) Shutdown(context.Context) error {
	e.stopped.Store(true)
	return nil
}

// ForceFlush performs no action.
func (*Exporter) ForceFlush(context.Context) error {
	return nil
}
