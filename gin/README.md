# go-bootstrap/gin

Gin middleware for OpenTelemetry context propagation.

## Installation

```bash
go get github.com/LinPr/go-bootstrap/gin
```

## Usage

`WithOtelBaggageFromHeader` extracts OpenTelemetry propagation data from HTTP
headers and turns the given header values into baggage members on the request
context.

```go
r := gin.New()
r.Use(otelgin.Middleware("my-service"))
r.Use(bsgin.WithOtelBaggageFromHeader("X-Request-Id", "X-User-Id"))
```

## Test

```bash
cd gin && go test ./...
```
