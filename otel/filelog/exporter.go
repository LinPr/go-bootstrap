package filelog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/LinPr/go-bootstrap/otel/filelog/transform"
	"go.opentelemetry.io/otel/sdk/log"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ log.Exporter = &Exporter{}

// ErrExporterShutdown is returned by [Exporter.Export] once
// [Exporter.Shutdown] has been called. sdk/log v0.14.0 keeps its equivalent
// sentinel unexported, so this package declares its own.
var ErrExporterShutdown = errors.New("exporter is shutdown")

// Exporter writes JSON-encoded log records to an [io.Writer] ([os.Stdout] by default).
// Exporter must be created with [New].
type Exporter struct {
	encoder atomic.Pointer[json.Encoder]
	writer  io.Writer
	format  LogFormat
	stopped atomic.Bool

	stdoutExporter log.Exporter
	// inst       *observ.Instrumentation
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)
	enc := json.NewEncoder(cfg.Writer)

	exporter := &Exporter{
		writer: cfg.Writer,
		format: cfg.Format,
	}

	exporter.encoder.Store(enc)

	return exporter, nil

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
	encoder := e.encoder.Load()
	if e.stopped.Load() {
		return ErrExporterShutdown
	}

	switch e.format {
	case LogFormatOtlpJson:

		data, err := processOtlpJson(records)
		if err != nil {
			return err
		}

		return encoder.Encode(&data)

	case LogFormatJson:
		results, err := processRecordJson(records)
		if err != nil {
			return err
		}
		for _, record := range results {
			if err := encoder.Encode(&record); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported log format: %s", e.format)
	}
}

// Shutdown shuts down the Exporter. Calls to Export after Shutdown return
// [ErrExporterShutdown].
func (e *Exporter) Shutdown(context.Context) error {
	// Store stopped first so Export cannot observe a cleared encoder while the
	// Exporter still appears active.
	e.stopped.Store(true)
	e.encoder.Store(nil)
	return nil
}

// ForceFlush performs no action.
func (*Exporter) ForceFlush(context.Context) error {
	return nil
}

func processOtlpJson(records []log.Record) (map[string]any, error) {
	resourceLogs := transformResourceLogs(records)
	if resourceLogs == nil {
		return nil, nil
	}

	pbRequest := &collogpb.ExportLogsServiceRequest{ResourceLogs: resourceLogs}

	b, err := protojson.MarshalOptions{
		UseEnumNumbers: true,
	}.Marshal(pbRequest)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	decodeSpanIDs(data)
	return data, nil
}

func processRecordJson(records []log.Record) ([]transform.RecordJSON, error) {
	results := make([]transform.RecordJSON, 0, len(records))
	for _, r := range records {
		results = append(results, transform.NewRecordJSON(r))
	}
	return results, nil
}
