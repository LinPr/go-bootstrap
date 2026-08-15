package http

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// NewHttpClientOtelTransport creates an HTTP transport with OpenTelemetry instrumentation
// The options passed to the fucntion will override the default options
func NewHttpClientOtelTransport(base http.RoundTripper, opts ...otelhttp.Option) *otelhttp.Transport {
	return otelhttp.NewTransport(
		base,
		append(
			[]otelhttp.Option{
				otelhttp.WithTracerProvider(otel.GetTracerProvider()),
				otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			},
			opts...,
		)...,
	)
}

// NewHttpServerOtelHandler wraps an HTTP handler with OpenTelemetry instrumentation
func NewHttpServerOtelHandler(handler http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	return otelhttp.NewHandler(handler, operation, opts...)
}
