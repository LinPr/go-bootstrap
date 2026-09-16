# go-bootstrap/otel

OpenTelemetry initialization toolkit for Log, Trace, and Metric. Provides a
unified `Config`, sensible defaults, pluggable exporters, and bridges for
popular Go logging libraries.

## Installation

```bash
go get github.com/LinPr/go-bootstrap/otel
```

## Quick Start

```go
package main

import (
	"context"
	"log"

	bsotel "github.com/LinPr/go-bootstrap/otel"
)

func main() {
	// Start from defaults and override only what you need.
	cfg := bsotel.DefaultConfig()
	cfg.ServiceName = "my-service"
	cfg.ServiceVersion = "1.0.0"

	// Initialize the global providers (log + trace + metric).
	if _, err := bsotel.NewOtelProviders(cfg); err != nil {
		log.Fatal(err)
	}
	defer bsotel.ShutdownOtelProvider(context.Background())
}
```

## Configuration

`Config` is merged on top of `DefaultConfig()`: any zero-valued field in the
passed `Config` falls back to the default, while non-zero fields override it.
This lets you set only the fields you care about.

- **bool fields** are always assigned directly (`false` is a valid override).
- **string / int / float64 fields** use zero-value fallback (empty/0 keeps the default).
- **map fields** fall back when `nil`.

```go
cfg := &bsotel.Config{
	ServiceName:    "go-bootstrap",
	ServiceVersion: "1.0.0",
	Log: bsotel.LogConfig{
		Enable:               true,
		Exporter:             bsotel.ExporterTypeHTTP,
		Logger:               bsotel.LoggerTypeSlog,
		Level:                "info",
		RemoteAddr:           "http://localhost:4318/v1/logs",
		Headers:              map[string]string{"X-Custom": "value"},
		Pretty:               true,
		AttributeCountLimit:  128,
		Rotate: bsotel.Rotate{
			Filename:   "./log/go-bootstrap.json",
			MaxMB:      1024,
			MaxDay:     1,
			MaxBackups: 7,
			LocalTime:  true,
			Compress:   true,
		},
	},
	Trace: bsotel.TraceConfig{
		Enable:        true,
		Exporter:      bsotel.ExporterTypeGRPC,
		RemoteAddr:    "localhost:4317",
		Headers:       map[string]string{"X-Custom": "value"},
		Pretty:        true,
		SamplingRatio: 1.0,
	},
	Metric: bsotel.MetricConfig{
		Enable:               true,
		Exporter:             bsotel.ExporterTypePrometheus,
		RemoteAddr:           "localhost:4317",
		Headers:              map[string]string{"X-Custom": "value"},
		Pretty:               true,
		IntervalSeconds:      10,
		EnableRuntimeMetrics: true,
		CardinalityLimit:     2000,
	},
}

if _, err := bsotel.NewOtelProviders(cfg); err != nil {
	log.Fatal(err)
}
defer bsotel.ShutdownOtelProvider(context.Background())
```

### Field reference

| Section | Field | Type | Description |
| --- | --- | --- | --- |
| `Config` | `ServiceName` | `string` | Service name |
| | `ServiceVersion` | `string` | Service version |
| `LogConfig` | `Enable` | `bool` | Toggle logging |
| | `Exporter` | `ExporterType` | `stdout` / `file` / `http` / `grpc` |
| | `Logger` | `LoggerType` | `slog` / `zap` / `logrus` / `logr` |
| | `Level` | `string` | `debug` / `info` / `warn` / `error` |
| | `RemoteAddr` | `string` | HTTP URL or gRPC address |
| | `Headers` | `map[string]string` | Request headers for HTTP/gRPC |
| | `Pretty` | `bool` | Pretty-print stdout / JSON attributes |
| | `AttributeCountLimit` | `int` | Max attributes per log record |
| | `Rotate` | `Rotate` | File rotation (see below) |
| `Rotate` | `Filename` | `string` | Log file path |
| | `MaxMB` | `int` | Max file size in MB before rotation |
| | `MaxDay` | `int` | Max days to retain old files |
| | `MaxBackups` | `int` | Max number of old files to keep |
| | `LocalTime` | `bool` | Use local time for backups |
| | `Compress` | `bool` | Gzip rotated files |
| `TraceConfig` | `Enable` | `bool` | Toggle tracing |
| | `Exporter` | `ExporterType` | `stdout` / `http` / `grpc` |
| | `RemoteAddr` | `string` | HTTP URL or gRPC address |
| | `Headers` | `map[string]string` | Request headers for HTTP/gRPC |
| | `Pretty` | `bool` | Pretty-print stdout |
| | `SamplingRatio` | `float64` | Sampling ratio (0.0–1.0) |
| `MetricConfig` | `Enable` | `bool` | Toggle metrics |
| | `Exporter` | `ExporterType` | `stdout` / `http` / `grpc` / `prometheus` |
| | `RemoteAddr` | `string` | HTTP URL or gRPC address |
| | `Headers` | `map[string]string` | Request headers for HTTP/gRPC |
| | `Pretty` | `bool` | Pretty-print stdout |
| | `IntervalSeconds` | `int` | Export interval in seconds |
| | `EnableRuntimeMetrics` | `bool` | Go runtime metrics |
| | `CardinalityLimit` | `int` | Max unique label combinations per instrument |

## Features

### Logger Bridges

Supports multiple popular Go logging libraries with automatic context
propagation. The selected bridge is wired into the global logger during
initialization:

- **slog**: Standard library structured logging
- **zap**: Uber's high-performance logger
- **logrus**: Classic structured logger
- **logr**: Generic logging interface

A severity floor is applied via a custom `severityProcessor` (and
`otelSeverityLogger` for the provider path) so records below the configured
`Level` are dropped before export.

### JSON attribute handler (slog)

When `Log.Pretty` is enabled with the slog bridge, a `jsonHandler` middleware
serializes struct/map/slice attribute values to JSON before the otelslog
bridge flattens them via `fmt %+v`. This preserves structured data in log
output.

### File log exporter with rotation

The `ExporterTypeFile` log exporter uses the `otlplogfile` package, which
writes OTLP-formatted JSON to a writer (default `os.Stdout`). When a
`Rotate.Filename` is set, output goes through [lumberjack](https://github.com/natefinch/lumberjack)
for size/age-based rotation with compression. `traceId` and `spanId` in the
output are converted from base64 to hex for readability.

### Context-Aware Logging with zapsugar

The `zapsugar` package provides a convenient wrapper for zap that automatically extracts span context from `context.Context`:

```go
import (
	"github.com/LinPr/go-bootstrap/otel/zapsugar"
	"go.uber.org/zap"
)

// Package-level functions
zapsugar.Infow(ctx, "user logged in", "user_id", 123)
zapsugar.Errorf(ctx, "failed to process request: %v", err)

// Scoped loggers for different modules (pass nil to use the global zap logger)
moduleLogger := zapsugar.NewSubScopedZapSugar("auth-module", nil)
moduleLogger.Infow(ctx, "authentication successful", "method", "oauth2")

// Nested scopes reuse an existing *zap.SugaredLogger
subLogger := zapsugar.NewSubScopedZapSugar("token-validator", moduleLogger.Logger())
subLogger.Debugw(ctx, "validating token", "issuer", "auth0")

// Add persistent attributes
requestLogger := moduleLogger.WithAttribute("request_id", "req-12345")
requestLogger.Infow(ctx, "processing request")

// Extract OpenTelemetry baggage members into log fields
baggageLogger := moduleLogger.WithBaggageMembers("user.id", "request.id")
baggageLogger.Infow(ctx, "handling request") // adds user.id / request.id if present in baggage
```

### Baggage-aware slog handler

The `slogbaggage` package wraps a `slog.Handler` so selected baggage members
are injected as log attributes:

```go
import (
	"log/slog"
	"os"

	"github.com/LinPr/go-bootstrap/otel/slogbaggage"
)

handler := slog.NewJSONHandler(os.Stdout, nil)
handler = slogbaggage.NewBaggageHandler(handler,
	slogbaggage.WithBaggageMembers("user.id", "request.id"),
)
slog.SetDefault(slog.New(handler))
```

### Context propagation helpers

The `propagation` package provides utilities to extract span context and
baggage from HTTP headers:

```go
import "github.com/LinPr/go-bootstrap/otel/propagation"

spanCtx, bag := propagation.ExtractPropagationsFromRequestHeader(r.Header)
```

It also offers `UnmarshalSpanContextConfig` to rebuild a `trace.SpanContextConfig` from JSON.

### HTTP and gRPC Instrumentation

Use the sibling `http` and `grpc` modules for instrumented transports and
interceptors:

```go
import "github.com/LinPr/go-bootstrap/http"

// HTTP server handler with tracing
handler := bshttp.NewHttpServerOtelHandler(mux, "my-service")

// HTTP client transport with tracing
client := &http.Client{
	Transport: bshttp.NewHttpClientOtelTransport(http.DefaultTransport),
}
```

```go
import "github.com/LinPr/go-bootstrap/grpc"

// gRPC server with baggage + logging interceptors
server := grpc.NewServer(
	grpc.ChainUnaryInterceptor(
		bsgrpc.UnaryServerBaggageInterceptor("x-request-id"),
		bsgrpc.UnaryServerLoggingInterceptor(),
	),
	grpc.StatsHandler(bsgrpc.NewServerMessageSizeStatsHandler()),
)
```

## Constants

### Exporters

- `ExporterTypeStdout` - Console output
- `ExporterTypeFile` - File output (log only, with rotation via lumberjack)
- `ExporterTypeHTTP` - OTLP over HTTP
- `ExporterTypeGRPC` - OTLP over gRPC
- `ExporterTypePrometheus` - Prometheus (metrics only)

### Logger Types

- `LoggerTypeSlog` - Standard library slog
- `LoggerTypeZap` - Uber zap
- `LoggerTypeLogrus` - Sirupsen logrus
- `LoggerTypeLogr` - Go-logr interface

## API

| Function | Description |
| --- | --- |
| `NewOtelProviders(config *Config) (*OtelProviders, error)` | Initialize providers and set them as global |
| `ShutdownOtelProvider(ctx context.Context) error` | Flush and shutdown the global providers |
| `GetOtelProvider() *OtelProviders` | Returns the global provider set |
| `(*OtelProviders).GetLoggerProvider()` | Returns the log provider |
| `(*OtelProviders).GetTracerProvider()` | Returns the trace provider |
| `(*OtelProviders).GetMeterProvider()` | Returns the metric provider |
| `DefaultConfig() *Config` | Returns the default configuration |

## Testing

```bash
cd otel
go test ./...
```
