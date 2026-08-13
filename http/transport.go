package http

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewOtelHttpTransport creates an HTTP transport with OpenTelemetry instrumentation
func NewOtelHttpTransport() *otelhttp.Transport {
	return otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
		}),
	)
}

// NewOtelHttpHandler wraps an HTTP handler with OpenTelemetry instrumentation
func NewOtelHttpHandler(handler http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	return otelhttp.NewHandler(handler, operation, opts...)
}
