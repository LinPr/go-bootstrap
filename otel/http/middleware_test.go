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

func TestServerLoggingMiddleware(t *testing.T) {
	handler := ServerLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response: " + string(body)))
	}))

	req := httptest.NewRequest(http.MethodPost, "/test?foo=bar", bytes.NewBufferString("test request body"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-client")

	previousLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

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
}

func TestLoggingTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("server response: " + string(body)))
	}))
	defer server.Close()

	previousLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	client := NewLoggingHttpClient()

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
}

func TestNewLoggingHttpClient(t *testing.T) {
	client := NewLoggingHttpClient()
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.Transport == nil {
		t.Fatal("transport is nil")
	}
}

func TestNewOtelLoggingHttpClient(t *testing.T) {
	client := NewOtelLoggingHttpClient()
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.Transport == nil {
		t.Fatal("transport is nil")
	}
}
