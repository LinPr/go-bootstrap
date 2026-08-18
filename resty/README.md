# go-bootstrap/resty

Resty v3 middleware for OpenTelemetry context propagation.

## Installation

```bash
go get github.com/LinPr/go-bootstrap/resty
```

## Usage

`WithOtelPropagationInjection` injects trace and baggage data from the request
context into outgoing HTTP headers.

```go
client := resty.New()
client.AddRequestMiddleware(bsresty.WithOtelPropagationInjection())
```

## Test

```bash
cd resty && go test ./...
```
