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
	otlpBuf bytes.Buffer  // reusable scratch for the otlp_json format
	otlpEnc *json.Encoder // encoder over otlpBuf
	buf     []byte        // reusable encoding buffer for the json format
}

// New creates an [Exporter].
func New(options ...Option) (*Exporter, error) {
	cfg := newConfig(options)

	exporter := &Exporter{
		writer: cfg.Writer,
		format: cfg.Format,
	}

	exporter.otlpBuf.Grow(64 << 10)
	exporter.otlpEnc = json.NewEncoder(&exporter.otlpBuf)

	return exporter, nil
}

var transformResourceLogs = transform.ResourceLogs

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
		return e.exportOtlpJson(records)
	case LogFormatJson:
		return e.exportJSON(records)
	default:
		return fmt.Errorf("unsupported log format: %s", e.format)
	}
}

// exportOtlpJson encodes the whole batch as one OTLP JSON envelope.
func (e *Exporter) exportOtlpJson(records []log.Record) error {
	data, err := processOtlpJson(records)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Encode into the reusable buffer first: json.Encoder writes straight
	// through to its io.Writer, and we want a single Write per batch (and none
	// at all if encoding fails partway).
	e.otlpBuf.Reset()
	if err := e.otlpEnc.Encode(data); err != nil {
		return err
	}
	if _, err := e.writer.Write(e.otlpBuf.Bytes()); err != nil {
		return err
	}
	return nil
}

// exportJSON writes one JSON object per record, newline-terminated, using a
// buffer reused across calls so a whole batch costs a single Write.
func (e *Exporter) exportJSON(records []log.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	buf := e.buf[:0]
	for i := range records {
		buf = transform.AppendRecordJSON(buf, records[i])
		buf = append(buf, '\n')
	}
	e.buf = buf

	if _, err := e.writer.Write(buf); err != nil {
		return err
	}
	return nil
}

// Shutdown shuts down the Exporter. Calls to Export after Shutdown return
// [log.ErrExporterShutdown].
func (e *Exporter) Shutdown(context.Context) error {
	// Store stopped first so Export cannot observe a released buffer while the
	// Exporter still appears active.
	e.stopped.Store(true)
	e.mu.Lock()
	e.buf = nil
	e.otlpEnc = nil
	e.otlpBuf.Reset()
	e.writer = io.Discard
	e.mu.Unlock()
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
