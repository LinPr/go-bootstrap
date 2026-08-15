package slogbaggage

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/baggage"
)

func TestBaggageHandler(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(NewBaggageHandler(jsonHandler, WithBaggageMembers("user.id", "request.id")))

	slog.SetDefault(logger)

	userID, _ := baggage.NewMember("user.id", "12345")
	requestID, _ := baggage.NewMember("request.id", "req-789")
	sessionID, _ := baggage.NewMember("session.id", "sess-abc")

	bag, err := baggage.New(userID, requestID, sessionID)
	if err != nil {
		t.Fatalf("failed to create baggage: %v", err)
	}

	ctx := baggage.ContextWithBaggage(context.Background(), bag)

	slog.InfoContext(ctx, "test message with baggage")

	t.Logf("Log output:\n%s", buf.String())

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v", err)
	}

	if logEntry["user.id"] != "12345" {
		t.Errorf("expected user.id=12345, got %v", logEntry["user.id"])
	}
	if logEntry["request.id"] != "req-789" {
		t.Errorf("expected request.id=req-789, got %v", logEntry["request.id"])
	}
	if logEntry["session.id"] != "sess-abc" {
		t.Errorf("expected session.id=sess-abc, got %v", logEntry["session.id"])
	}
}
