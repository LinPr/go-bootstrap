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

// ClientOption configures an HTTP client
type ClientOption func(*http.Client)

// WithOtelHttpTransport enables OTEL instrumentation for HTTP client
func WithOtelHttpTransport() ClientOption {
	return func(c *http.Client) {
		if c.Transport == nil {
			c.Transport = NewOtelHttpTransport()
		} else {
			c.Transport = NewOtelHttpTransport()
		}
	}
}

// WithClientDebugLog enables debug logging for HTTP client requests and responses
func WithClientDebugLog() ClientOption {
	return func(c *http.Client) {
		if c.Transport == nil {
			c.Transport = http.DefaultTransport
		}
		c.Transport = &loggingRoundTripper{next: c.Transport}
	}
}

// NewHttpClient creates an HTTP client with the given options
func NewHttpClient(opts ...ClientOption) *http.Client {
	client := &http.Client{
		Transport: http.DefaultTransport,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// ServerOption configures an HTTP server handler
type ServerOption func(http.Handler) http.Handler

// WithServerDebugLog enables debug logging for HTTP server requests and responses
func WithServerDebugLog() ServerOption {
	return func(next http.Handler) http.Handler {
		return ServerLoggingMiddleware(next)
	}
}

// NewHttpHandler wraps an HTTP handler with the given options
func NewHttpHandler(handler http.Handler, opts ...ServerOption) http.Handler {
	for _, opt := range opts {
		handler = opt(handler)
	}
	return handler
}
