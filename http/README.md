# go-bootstrap/http

Standard `net/http` middleware and OpenTelemetry-instrumented client/server helpers.

## Installation

```bash
go get github.com/LinPr/go-bootstrap/http
```

## Features

- Middleware chaining: `Chain`, `NewHandlerChain`
- OpenTelemetry: `OtelMiddleware`, `NewHttpServerOtelHandler`, `NewHttpClientOtelTransport`
- Debug logging: `WithServerDebugLog`, `WithClientDebugLog` (logs equivalent curl commands)
- Client builder: `NewHttpClient` with functional options

## Usage

```go
handler := bshttp.Chain(mux,
	bshttp.OtelMiddleware("my-service"),
	bshttp.WithServerDebugLog,
)

client := bshttp.NewHttpClient(
	bshttp.WithOtelHttpTransport(nil),
	bshttp.WithClientDebugLog(),
)
```

## Test

```bash
cd http && go test ./...
```
