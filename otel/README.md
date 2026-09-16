# go-bootstrap/otel

OpenTelemetry initialization toolkit for Log, Trace, and Metric.

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
	cfg := bsotel.DefaultConfig()
	cfg.ServiceName = "my-service"
	cfg.ServiceVersion = "1.0.0"

	if err := bsotel.InitOtelProvider(cfg); err != nil {
		log.Fatal(err)
	}
	defer bsotel.ShutdownOtelProvider(context.Background())
}
```

## Configuration

`Config` is merged on top of `DefaultConfig()`: any zero-valued field in the
passed `Config` falls back to the default, while non-zero fields override it.
This lets you set only the fields you care about.

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

Supports multiple popular Go logging libraries with automatic context propagation:

- **slog**: Standard library structured logging
- **zap**: Uber's high-performance logger
- **logrus**: Classic structured logger
- **logr**: Generic logging interface

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

### HTTP and gRPC Instrumentation

```go
import "github.com/LinPr/go-bootstrap/otel/transport"

// HTTP client with tracing
client := &http.Client{
	Transport: transport.NewOtelHttpTransport(),
}

// gRPC client with tracing
conn, err := grpc.NewClient(
	"localhost:50051",
	transport.WithOtelGRPCClientOption(),
)

// gRPC server with tracing
server := grpc.NewServer(
	transport.WithOtelGRPCServerOption(),
)
```

## Constants

### Exporters

- `ExporterTypeStdout` - Console output
- `ExporterTypeFile` - File output (log only, with rotation)
- `ExporterTypeHTTP` - OTLP over HTTP
- `ExporterTypeGRPC` - OTLP over gRPC
- `ExporterTypePrometheus` - Prometheus (metrics only)

### Logger Types

- `LoggerTypeSlog` - Standard library slog
- `LoggerTypeZap` - Uber zap
- `LoggerTypeLogrus` - Sirupsen logrus
- `LoggerTypeLogr` - Go-logr interface

## Testing

```bash
cd otel
go test ./...
```
