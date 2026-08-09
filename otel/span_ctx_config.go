package otel

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

func UnmarshalSpanContextConfig(data []byte, spanCtxCfg *trace.SpanContextConfig) error {
	if spanCtxCfg == nil {
		return fmt.Errorf("unmarshal span context config: nil target")
	}

	if strings.TrimSpace(string(data)) == "null" {
		*spanCtxCfg = trace.SpanContextConfig{}
		return nil
	}

	var raw struct {
		TraceID    string `json:"TraceID"`
		SpanID     string `json:"SpanID"`
		TraceFlags string `json:"TraceFlags"`
		TraceState string `json:"TraceState"`
		Remote     bool   `json:"Remote"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal span context config: %w", err)
	}

	traceID, err := trace.TraceIDFromHex(raw.TraceID)
	if err != nil {
		return fmt.Errorf("unmarshal span context config trace id: %w", err)
	}

	spanID, err := trace.SpanIDFromHex(raw.SpanID)
	if err != nil {
		return fmt.Errorf("unmarshal span context config span id: %w", err)
	}

	traceFlagsBytes, err := hex.DecodeString(raw.TraceFlags)
	if err != nil {
		return fmt.Errorf("unmarshal span context config trace flags: %w", err)
	}
	if len(traceFlagsBytes) != 1 {
		return fmt.Errorf("unmarshal span context config trace flags: expected 1 byte, got %d", len(traceFlagsBytes))
	}

	traceState, err := trace.ParseTraceState(raw.TraceState)
	if err != nil {
		return fmt.Errorf("unmarshal span context config trace state: %w", err)
	}

	*spanCtxCfg = trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(traceFlagsBytes[0]),
		TraceState: traceState,
		Remote:     raw.Remote,
	}

	return nil
}
