package otlplogfile

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
	buf     bytes.Buffer
	stopped atomic.Bool
	// timestamps bool
	// inst       *observ.Instrumentation
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)

	e := &Exporter{
		writer: cfg.Writer,
	}

	var err error
	// e.inst, err = observ.NewInstrumentation(counter.NextExporterID())
	return e, err
}

var transformResourceLogs = transform.ResourceLogs

func (e *Exporter) Export(ctx context.Context, records []log.Record) error {
	defer e.buf.Reset()
	if ctx.Err() != nil {
		return ctx.Err()
	}

	resourceLogs := transformResourceLogs(records)
	if resourceLogs == nil {
		return nil
	}

	pbRequest := &collogpb.ExportLogsServiceRequest{ResourceLogs: resourceLogs}

	b, err := protojson.MarshalOptions{}.Marshal(pbRequest)
	if err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	convertTraceIDs(data)

	b, err = json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err = e.buf.Write(b); err != nil {
		return err
	}

	if err := e.buf.WriteByte('\n'); err != nil {
		return err
	}

	if _, err := e.writer.Write(e.buf.Bytes()); err != nil {
		return err
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
