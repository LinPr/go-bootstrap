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

This is a Go workspace (`go.work`) holding several independent modules, so
package patterns like `./...` do not work from the repository root. Use the
`Taskfile.yml` tasks, which pick the Go toolchain matching the checked-out
branch's `go.work`:

```bash
task env        # branch, required Go version, actual toolchain
task build      # build every workspace module
task vet        # vet every workspace module
task test       # all tests, including the integration module
task test:unit  # library modules only
task tidy       # go mod tidy in each module
task ci         # vet + test
```

Do not run `go work sync` here: it rewrites every module's `go.mod` and
silently upgrades dependencies.
