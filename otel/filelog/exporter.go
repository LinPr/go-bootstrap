package filelog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/LinPr/go-bootstrap/otel/filelog/transform"
	"go.opentelemetry.io/otel/sdk/log"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ log.Exporter = &Exporter{}

// Exporter writes JSON-encoded log records to an [io.Writer] ([os.Stdout] by
// default).
// Exporter must be created with [New].
type Exporter struct {
	writer  io.Writer
	format  LogFormat
	stopped atomic.Bool

	// mu serializes Export so concurrent callers cannot interleave their
	// records, and guards the reusable buffers below.
	mu      sync.Mutex
	buf     bytes.Buffer  // reusable scratch for the otlp_json format
	jsonEnc *json.Encoder // encoder over otlpBuf
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)

	exporter := &Exporter{
		writer: cfg.Writer,
		format: cfg.Format,
	}

	exporter.buf.Grow(64 << 10)
	exporter.jsonEnc = json.NewEncoder(&exporter.buf)

	return exporter, nil
}

// Export handles the delivery of log records to external receivers.
// Any of the Exporter's methods may be called concurrently with itself or with
// other methods. It is the responsibility of the Exporter to manage this
// concurrency.
func (e *Exporter) Export(ctx context.Context, records []log.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.stopped.Load() {
		return log.ErrExporterShutdown
	}
	if len(records) == 0 {
		return nil
	}

	switch e.format {
	case LogFormatOtlpJson:
		data, err := processOtlpJson(records)

		if err != nil {
			return err
		}
		e.mu.Lock()
		defer e.mu.Unlock()

		e.buf.Reset()
		if err := e.jsonEnc.Encode(data); err != nil {
			return err
		}
		if _, err := e.writer.Write(e.buf.Bytes()); err != nil {
			return err
		}
		return nil
	case LogFormatJson:
		recordJsons := transform.RecordJSONs(records)

		e.mu.Lock()
		defer e.mu.Unlock()
		e.buf.Reset()

		for _, recordJson := range recordJsons {
			if err := e.jsonEnc.Encode(recordJson); err != nil {
				return err
			}
		}

		if _, err := e.writer.Write(e.buf.Bytes()); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unsupported log format: %s", e.format)
	}
}

// Shutdown shuts down the Exporter. Calls to Export after Shutdown return
// [log.ErrExporterShutdown].
func (e *Exporter) Shutdown(context.Context) error {
	// Store stopped first so Export cannot observe a released buffer while the
	// Exporter still appears active.
	e.stopped.Store(true)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.jsonEnc = nil
	e.buf.Reset()
	e.writer = io.Discard
	return nil
}

// ForceFlush performs no action.
func (*Exporter) ForceFlush(context.Context) error {
	return nil
}

func processOtlpJson(records []log.Record) (map[string]any, error) {
	resourceLogs := transform.ResourceLogs(records)
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
