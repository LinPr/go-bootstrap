package resty

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	restyv3 "resty.dev/v3"
)

// WithOtelPropagationInjection serializes the Trace information and Baggage data
// from the current Context and injects them into the HTTP Header.
func WithOtelPropagationInjection() restyv3.RequestMiddleware {
	return func(c *restyv3.Client, req *restyv3.Request) error {
		ctx := req.Context()
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		return nil
	}
}
