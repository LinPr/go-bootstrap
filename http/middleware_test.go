package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestLogger(t *testing.T) {
	previousLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})
}

func TestServerMiddlewares(t *testing.T) {
	setupTestLogger(t)

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response: " + string(body)))
	})

	t.Run("debug log middleware", func(t *testing.T) {
		handler := Chain(baseHandler, WithServerDebugLog)

		req := httptest.NewRequest(http.MethodPost, "/test?foo=bar", bytes.NewBufferString("test request body"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-client")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", recorder.Code)
		}

		body := recorder.Body.String()
		expected := "response: test request body"
		if body != expected {
			t.Errorf("expected body %q, got %q", expected, body)
		}
	})

	t.Run("otel and debug log middleware", func(t *testing.T) {
		handler := Chain(baseHandler, OtelMiddleware("test-operation"), WithServerDebugLog)

		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString("otel test body"))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", recorder.Code)
		}

		body := recorder.Body.String()
		expected := "response: otel test body"
		if body != expected {
			t.Errorf("expected body %q, got %q", expected, body)
		}
	})
}

func TestClientMiddlewares(t *testing.T) {
	setupTestLogger(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("server response: " + string(body)))
	}))
	defer server.Close()

	t.Run("debug log only", func(t *testing.T) {
		client := NewHttpClient(WithClientDebugLog())

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/api/test", bytes.NewBufferString("client request"))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "server response: client request"
		if string(body) != expected {
			t.Errorf("expected body %q, got %q", expected, string(body))
		}
	})

	t.Run("otel and debug log", func(t *testing.T) {
		client := NewHttpClient(WithOtelHttpTransport(nil), WithClientDebugLog())

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/api/otel", bytes.NewBufferString("otel request"))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "server response: otel request"
		if string(body) != expected {
			t.Errorf("expected body %q, got %q", expected, string(body))
		}
	})
}

func TestMiddlewareExecutionOrder(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-after")
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware3-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware3-after")
		})
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Chain execution order", func(t *testing.T) {
		executionOrder = []string{}
		handler := Chain(baseHandler, middleware1, middleware2, middleware3)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		expected := []string{
			"middleware1-before",
			"middleware2-before",
			"middleware3-before",
			"handler",
			"middleware3-after",
			"middleware2-after",
			"middleware1-after",
		}

		if len(executionOrder) != len(expected) {
			t.Fatalf("expected %d steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
		}

		for i, step := range expected {
			if executionOrder[i] != step {
				t.Errorf("step %d: expected %q, got %q", i, step, executionOrder[i])
			}
		}
	})

	t.Run("Use method execution order", func(t *testing.T) {
		executionOrder = []string{}
		handler := NewHandlerChain(baseHandler).
			Use(middleware1).
			Use(middleware2, middleware3).
			Build()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		expected := []string{
			"middleware1-before",
			"middleware2-before",
			"middleware3-before",
			"handler",
			"middleware3-after",
			"middleware2-after",
			"middleware1-after",
		}

		if len(executionOrder) != len(expected) {
			t.Fatalf("expected %d steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
		}

		for i, step := range expected {
			if executionOrder[i] != step {
				t.Errorf("step %d: expected %q, got %q", i, step, executionOrder[i])
			}
		}
	})
}
