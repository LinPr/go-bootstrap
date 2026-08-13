package http

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
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

func buildCurlCommand(r *http.Request, body []byte) string {
	var curl strings.Builder
	curl.WriteString("curl -X ")
	curl.WriteString(r.Method)

	for key, values := range r.Header {
		for _, value := range values {
			curl.WriteString(fmt.Sprintf(" -H '%s: %s'", key, value))
		}
	}

	if len(body) > 0 {
		curl.WriteString(fmt.Sprintf(" -d '%s'", string(body)))
	}

	curl.WriteString(" '")
	curl.WriteString(r.URL.String())
	curl.WriteString("'")

	return curl.String()
}

// ServerLoggingMiddleware logs HTTP request and response details for server
func ServerLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		msg := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		if r.URL.RawQuery != "" {
			msg += "?" + r.URL.RawQuery
		}

		slog.DebugContext(ctx, msg,
			"status", recorder.statusCode,
			"request_header", r.Header,
			"request_body", string(reqBody),
			"response_header", recorder.Header(),
			"response_body", recorder.body.String(),
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
	ctx := req.Context()

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	curlCmd := buildCurlCommand(req, reqBody)

	resp, err := t.next.RoundTrip(req)

	if err != nil {
		msg := fmt.Sprintf("%s %s", req.Method, req.URL.String())
		slog.ErrorContext(ctx, msg,
			"request_header", req.Header,
			"request_body", string(reqBody),
			"curl_cmd", curlCmd,
			"error", err,
		)
		return nil, err
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	msg := fmt.Sprintf("%s %s", req.Method, req.URL.String())
	slog.DebugContext(ctx, msg,
		"status", resp.StatusCode,
		"request_header", req.Header,
		"request_body", string(reqBody),
		"response_header", resp.Header,
		"response_body", string(respBody),
		"curl_cmd", curlCmd,
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
