# go-bootstrap/otel

OpenTelemetry for logs, traces, and metrics: one `Config`, pluggable exporters,
and logger bridges for `slog` and `zap`.

Subpackages: `otel/filelog` (file exporter with rotation), `otel/kv` (typed
attribute values), `otel/propagation` (span context / baggage from headers),
`otel/slogbaggage` (baggage as slog attributes), `otel/zapsugar`
(context-aware `zap.SugaredLogger`).

## Install

```bash
go get github.com/LinPr/go-bootstrap/otel
```

## Quick start

```go
import (
	"context"
	"log"

	bsotel "github.com/LinPr/go-bootstrap/otel"
)

cfg := bsotel.DefaultConfig()
cfg.ServiceName = "my-service"
cfg.Log.Logger = bsotel.LoggerTypeSlog

// Registers the global log / trace / metric providers.
if _, err := bsotel.NewOtelProviders(cfg); err != nil {
	log.Fatal(err)
}
defer bsotel.ShutdownOtelProvider(context.Background())
```

## Configuration

Always start from `DefaultConfig()`: `bool` fields (`Enable`, `Pretty`,
`LocalTime`, `Compress`, `EnableRuntimeMetrics`) are assigned directly, so a
zero-value `Config{}` disables every signal. Strings/ints use zero-value
fallback, `Headers` are merged key-wise, and `ResourceAttributes` is replaced
wholesale.

Defaults: all signals on with the `stdout` exporter, level `info`,
`Pretty: true`, sampling ratio `1.0`, metric interval `30s`, runtime metrics on,
`CardinalityLimit: 2000`.

```go
cfg := bsotel.DefaultConfig()
cfg.ServiceName = "go-bootstrap"
cfg.ServiceVersion = "1.0.0"
cfg.ResourceAttributes = []bsotel.AttributekeyValue{
	{Key: "deployment.environment", Value: "production"},
}

cfg.Log.Exporter = bsotel.ExporterTypeHTTP
cfg.Log.RemoteAddr = "http://localhost:4318/v1/logs"
cfg.Log.Logger = bsotel.LoggerTypeSlog

cfg.Trace.Exporter = bsotel.ExporterTypeGRPC
cfg.Trace.RemoteAddr = "localhost:4317"
cfg.Trace.SamplingRatio = 1.0

cfg.Metric.Exporter = bsotel.ExporterTypeHTTP
cfg.Metric.RemoteAddr = "http://localhost:4318/v1/metrics"
cfg.Metric.IntervalSeconds = 10
```

### Field reference

| Section | Field | Type | Description |
| --- | --- | --- | --- |
| `Config` | `ServiceName` / `ServiceVersion` | `string` | Service identity |
| | `ResourceAttributes` | `[]AttributekeyValue` | Extra resource attributes |
| `LogConfig` | `Enable` | `bool` | Toggle logging |
| | `Exporter` | `ExporterType` | `stdout` / `file` / `file_json` / `file_otlp_json` / `http` / `grpc` |
| | `Logger` | `LoggerType` | `slog` / `zap` |
| | `Level` | `string` | `debug` / `info` / `warn` / `error` |
| | `RemoteAddr`, `Headers` | `string`, `map[string]string` | Exporter endpoint and request headers |
| | `Pretty` | `bool` | Pretty-print output / JSON attributes |
| | `AttributeCountLimit` | `int` | Max attributes per record |
| | `Rotate` | `Rotate` | File rotation, see below |
| `Rotate` | `Filename` | `string` | Target path; empty or `stdout` writes to stdout |
| | `MaxMB`, `MaxDay`, `MaxBackups` | `int` | Size limit, retention days, backup count |
| | `LocalTime`, `Compress` | `bool` | Local timestamps, gzip rotated files |
| `TraceConfig` | `Enable`, `Pretty` | `bool` | Toggle tracing / pretty output |
| | `Exporter` | `ExporterType` | `stdout` / `http` / `grpc` |
| | `RemoteAddr`, `Headers` | `string`, `map[string]string` | Exporter endpoint and request headers |
| | `SamplingRatio` | `float64` | 0.0–1.0 |
| `MetricConfig` | `Enable`, `Pretty` | `bool` | Toggle metrics / pretty output |
| | `Exporter` | `ExporterType` | `stdout` / `http` / `grpc` (Prometheus reader is always added) |
| | `RemoteAddr`, `Headers` | `string`, `map[string]string` | Exporter endpoint and request headers |
| | `IntervalSeconds` | `int` | Export interval |
| | `EnableRuntimeMetrics` | `bool` | Go runtime metrics |
| | `CardinalityLimit` | `int` | Max label combinations per instrument |

### Resource attributes

Merged into the resource after `resource.WithFromEnv()` and
`service.name`/`service.version`, so configured values win (last value wins on
duplicate keys). Values go through `otel/kv.FromNameValue`; unsupported types
are stringified with `fmt.Sprint`.

## Features

### Logger bridges

`LoggerTypeSlog` and `LoggerTypeZap` are fully supported and install a global
logger. `LoggerTypeLogrus` / `LoggerTypeLogr` are accepted but currently
**no-ops**; any other value fails with `unsupported logger type`.

- slog: created with `otelslog.WithSource(true)` (`code.*` attributes); a
  `severityProcessor` drops records below `Level`.
- zap: `zap.IncreaseLevel`, `zap.AddCaller()`, stack traces at `Error`.
- With `Log.Pretty`, a `jsonHandler` middleware serializes composite
  struct/map/slice attribute values as JSON (structs lose unexported fields;
  `error` values pass through).

### File exporter with rotation

`ExporterTypeFile` / `ExporterTypeFileOtlpJson` write an OTLP JSON envelope;
`ExporterTypeFileJson` writes one flat JSON object per line (`trace_id` /
`span_id` in hex, all resource/scope/record attributes inlined). When
`Rotate.Filename` is a real path, [lumberjack](https://github.com/natefinch/lumberjack)
handles size/age rotation; empty or `stdout` writes to stdout.

```go
cfg.Log.Exporter = bsotel.ExporterTypeFileJson
cfg.Log.Rotate = bsotel.Rotate{
	Filename:   "./log/app.json",
	MaxMB:      100,
	MaxDay:     7,
	MaxBackups: 5,
	Compress:   true,
}
```

The exporter is safe for concurrent use and returns
`log.ErrExporterShutdown` after `Shutdown`.

### Metrics

Each metric provider gets a Prometheus reader (always registered; serve it
yourself with `promhttp.Handler()`) plus a periodic OTLP reader built from
`Exporter` / `RemoteAddr` / `IntervalSeconds`. `Exemplars` use
`exemplar.TraceBasedFilter`. File exporters are rejected for metrics.

### Context-aware logging (zapsugar)

`zapsugar` extracts span context from `context.Context` and passes it to the
`otelzap` core, so trace correlation only works when `Log.Logger` is
`LoggerTypeZap`.

```go
zapsugar.Infow(ctx, "user logged in", "user_id", 123)

logger := zapsugar.NewSubScopedZapSugar("auth", nil) // nil → global zap logger
logger = logger.WithAttribute("request_id", "req-123")
logger.WithBaggageMembers("user.id").Infow(ctx, "handling request")
```

`WithAttribute`, `WithBaggageMembers`, and `WithOptions` mutate the receiver.

### Baggage-aware slog handler

```go
handler := slogbaggage.NewBaggageHandler(
	slog.NewJSONHandler(os.Stdout, nil),
	slogbaggage.WithBaggageMembers("user.id", "request.id"),
)
slog.SetDefault(slog.New(handler))
```

### Propagation helpers

Initialization installs a composite propagator (`tracecontext` + `baggage`), so
W3C headers are handled by the `http` / `grpc` modules. For manual extraction:

```go
spanCtx, bag := propagation.ExtractPropagationsFromRequestHeader(r.Header)
```

`propagation.UnmarshalSpanContextConfig` rebuilds a `trace.SpanContextConfig`
from JSON.

### HTTP and gRPC instrumentation

See the sibling `http` and `grpc` modules: `NewHttpServerOtelHandler`,
`NewHttpClientOtelTransport`, `Chain`, `OtelMiddleware`, and gRPC
`UnaryServerBaggageInterceptor` / `UnaryServerLoggingInterceptor` /
`NewServerMessageSizeStatsHandler`.

## Constants

Exporters: `ExporterTypeStdout`, `ExporterTypeFile` (= `ExporterTypeFileOtlpJson`,
OTLP JSON envelope), `ExporterTypeFileJson` (flat JSON lines), `ExporterTypeHTTP`,
`ExporterTypeGRPC`. There is no Prometheus exporter constant — a Prometheus
reader is always registered.

Logger types: `LoggerTypeSlog`, `LoggerTypeZap` (implemented);
`LoggerTypeLogrus`, `LoggerTypeLogr` (reserved, no-op).

## Notes

- **TLS**: every OTLP exporter uses `WithInsecure()`, so `https://` addresses are
  still plaintext. Terminate TLS in the collector/sidecar or build your own
  exporter. `RemoteAddr` is a full URL for HTTP
  (`http://host:4318/v1/logs`) and `host:port` for gRPC (`host:4317`).
- **File exporter**: log-only. `Pretty` does not indent file output; it only
  controls the slog `jsonHandler` (which affects every exporter).

## Helpers

`otel/kv` converts `any` to `attribute.KeyValue` (ints, uints, floats, bool,
string; anything else via `fmt.Sprint`; `uint`/`uint64` become decimal strings):

```go
kv.FromNameValue("http.status_code", 200) // attribute.Int
kv.FromNameValue("ratio", 0.5)            // attribute.Float64
```

`otel/filelog` can be used standalone with any writer:

```go
exporter, err := filelog.New(
	filelog.WithWriter(os.Stdout),
	filelog.WithFormat(filelog.LogFormatJson), // default: LogFormatOtlpJson
)
```

## API

| Function | Description |
| --- | --- |
| `NewOtelProviders(*Config) (*OtelProviders, error)` | Initialize and register global providers |
| `ShutdownOtelProvider(ctx context.Context) error` | Flush and shut down global providers |
| `GetOtelProvider() *OtelProviders` | Global provider set |
| `(*OtelProviders).Shutdown(ctx context.Context) error` | Shut down a specific provider set |
| `(*OtelProviders).GetLoggerProvider()` / `GetTracerProvider()` / `GetMeterProvider()` | Signal providers |
| `DefaultConfig() *Config` | Default configuration |
| `kv.FromNameValue(key string, value any)` | Typed `attribute.KeyValue` |
| `propagation.ExtractPropagationsFromRequestHeader(http.Header)` | Span context + baggage |
| `propagation.UnmarshalSpanContextConfig([]byte, *trace.SpanContextConfig)` | Rebuild span context config from JSON |

## Testing

```bash
cd otel && go test ./...
cd test && go test ./filelog/ ./http/ ./grpc/   # integration tests
```
