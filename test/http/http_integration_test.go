package test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httppkg "github.com/LinPr/go-bootstrap/http"
	"github.com/LinPr/go-bootstrap/otel"
	"github.com/LinPr/go-bootstrap/test/config"
	otelgo "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func TestHttpOtelIntegration(t *testing.T) {
	_ = config.SetupTestOtelProvider(t)

	meter := otelgo.Meter("go-bootstrap-http-integration")
	metricPrefix := "go.bootstrap.http.integration."
	requestCounter, err := meter.Int64Counter(metricPrefix + "request.counter")
	if err != nil {
		t.Fatalf("failed to create request counter: %v", err)
	}
	histogram, err := meter.Float64Histogram(metricPrefix + "latency.ms")
	if err != nil {
		t.Fatalf("failed to create latency histogram: %v", err)
	}

	tracer := otelgo.Tracer("http-integration-test-tracer")

	var httpServerTraceID string
	httpServer := httptest.NewServer(
		httppkg.NewHttpServerOtelHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverCtx := r.Context()
			httpServerTraceID = trace.SpanContextFromContext(serverCtx).TraceID().String()
			slog.InfoContext(serverCtx, "HTTP server received request", "path", r.URL.Path, "method", r.Method)
			requestCounter.Add(serverCtx, 1, metric.WithAttributes(attribute.String("transport", "http-server")))
			histogram.Record(serverCtx, 12.5, metric.WithAttributes(attribute.String("transport", "http-server")))
			_, _ = w.Write([]byte("ok"))
		}), "http-server"))
	defer httpServer.Close()

	httpClient := &http.Client{Transport: httppkg.NewHttpClientOtelTransport(nil)}

	const iterations = 5
	for i := range iterations {
		rootCtx, rootSpan := tracer.Start(t.Context(), fmt.Sprintf("http-integration-root-%d", i))
		rootTraceID := rootSpan.SpanContext().TraceID().String()

		slog.InfoContext(rootCtx, "Starting HTTP integration test iteration", "iteration", i, "trace_id", rootTraceID)

		httpCtx, httpSpan := tracer.Start(rootCtx, "http-client-call")
		httpRequest, err := http.NewRequestWithContext(httpCtx, http.MethodGet, httpServer.URL+"/ping", nil)
		if err != nil {
			t.Fatalf("iteration %d: failed to create http request: %v", i, err)
		}
		slog.InfoContext(httpCtx, "Sending HTTP request", "iteration", i, "url", httpServer.URL+"/ping")
		httpResponse, err := httpClient.Do(httpRequest)
		if err != nil {
			t.Fatalf("iteration %d: http request failed: %v", i, err)
		}
		_ = httpResponse.Body.Close()
		if httpResponse.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: unexpected http status: %d", i, httpResponse.StatusCode)
		}
		requestCounter.Add(httpCtx, 1, metric.WithAttributes(attribute.String("transport", "http-client")))
		histogram.Record(httpCtx, 8.25, metric.WithAttributes(attribute.String("transport", "http-client")))
		httpSpan.End()

		requestCounter.Add(rootCtx, 1, metric.WithAttributes(attribute.String("transport", "root")))
		histogram.Record(rootCtx, 5.0, metric.WithAttributes(attribute.String("transport", "root")))

		rootSpan.End()
		slog.InfoContext(rootCtx, "Completed HTTP integration test iteration", "iteration", i)

		if i == 0 {
			if httpServerTraceID == "" {
				t.Fatal("http server trace id is empty")
			}
			if httpServerTraceID != rootTraceID {
				t.Fatalf("expected http trace id %s, got %s", rootTraceID, httpServerTraceID)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
		t.Logf("warning: failed to shutdown provider: %v", err)
	}
}
