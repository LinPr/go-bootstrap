package gin

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
)

// WithOtelBaggageFromHeader extracts OpenTelemetry propagation data from HTTP headers
// and creates baggage members from specified header values.
func WithOtelBaggageFromHeader(header ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)
		members := baggage.FromContext(ctx).Members()

		for _, h := range header {
			if v := c.Request.Header.Get(h); v != "" {
				m, err := baggage.NewMember(h, v)
				if err != nil {
					slog.Error("failed to create baggage member: " + err.Error())
					c.Next()
					return
				}
				members = append(members, m)
			}
		}

		newBaggage, err := baggage.New(members...)
		if err != nil {
			slog.Error("failed to create baggage: " + err.Error())
			c.Next()
			return
		}

		ctx = baggage.ContextWithBaggage(ctx, newBaggage)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(c.Request.Header))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
