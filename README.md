# go-bootstrap

A collection of independent Go modules providing OpenTelemetry-instrumented
building blocks and utilities for common frameworks.

## Modules

| Module | Description | README |
| ------ | ----------- | ------ |
| `otel` | OpenTelemetry initialization for Log, Trace, and Metric | [otel/README.md](otel/README.md) |
| `gin` | Gin middleware for OpenTelemetry baggage propagation | [gin/README.md](gin/README.md) |
| `grpc` | gRPC interceptors and stats handlers (baggage, logging, metrics) | [grpc/README.md](grpc/README.md) |
| `http` | `net/http` middleware and instrumented client/server helpers | [http/README.md](http/README.md) |
| `resty` | Resty v3 middleware for OpenTelemetry propagation | [resty/README.md](resty/README.md) |
| `task` | Generic concurrent task pool | [task/README.md](task/README.md) |

Each module is versioned and importable independently:

```bash
go get github.com/LinPr/go-bootstrap/<module>
```

## Development

This is a Go workspace (`go.work`). To work on all modules together:

```bash
go work sync
go test ./...
```
