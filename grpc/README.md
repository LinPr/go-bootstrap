# go-bootstrap/grpc

gRPC interceptors and stats handlers for OpenTelemetry baggage propagation,
logging, and message-size metrics.

## Installation

```bash
go get github.com/LinPr/go-bootstrap/grpc
```

## Features

- Baggage interceptors: `UnaryServerBaggageInterceptor`, `StreamServerBaggageInterceptor`
- Logging interceptors: unary/stream for both client and server
- Message-size stats handlers: `NewServerMessageSizeStatsHandler`, `NewClientMessageSizeStatsHandler`
- Metadata carrier helpers: `Inject`, `Extract`

## Usage

```go
server := grpc.NewServer(
	grpc.ChainUnaryInterceptor(
		bsgrpc.UnaryServerBaggageInterceptor("x-request-id"),
		bsgrpc.UnaryServerLoggingInterceptor(),
	),
	grpc.StatsHandler(bsgrpc.NewServerMessageSizeStatsHandler()),
)
```

## Test

```bash
cd grpc && go test ./...
```
