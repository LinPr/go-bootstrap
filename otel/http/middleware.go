package http

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// ServerLoggingMiddleware logs HTTP request and response details for server
func ServerLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		slog.DebugContext(ctx, "HTTP server request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"content_length", r.ContentLength,
			"headers", r.Header,
			"body", string(reqBody),
		)

		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		slog.DebugContext(ctx, "HTTP server response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", duration.Milliseconds(),
			"response_size", recorder.body.Len(),
			"headers", recorder.Header(),
			"body", recorder.body.String(),
		)
	})
}

type loggingRoundTripper struct {
	next http.RoundTripper
}

// NewLoggingTransport creates an HTTP client transport that logs requests and responses
func NewLoggingTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &loggingRoundTripper{next: next}
}

func (t *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	ctx := req.Context()

	reqDump, _ := httputil.DumpRequestOut(req, true)
	slog.DebugContext(ctx, "HTTP client request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
		"dump", string(reqDump),
	)

	resp, err := t.next.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		slog.ErrorContext(ctx, "HTTP client request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return nil, err
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	slog.DebugContext(ctx, "HTTP client response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
		"content_length", resp.ContentLength,
		"headers", resp.Header,
		"body", string(respBody),
	)

	return resp, nil
}

// NewLoggingHttpClient creates an HTTP client with logging transport
func NewLoggingHttpClient() *http.Client {
	return &http.Client{
		Transport: NewLoggingTransport(http.DefaultTransport),
	}
}

// NewOtelLoggingHttpClient creates an HTTP client with both OTEL instrumentation and logging
func NewOtelLoggingHttpClient() *http.Client {
	otelTransport := NewOtelHttpTransport()
	return &http.Client{
		Transport: NewLoggingTransport(otelTransport),
	}
}
