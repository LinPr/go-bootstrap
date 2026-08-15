package otel

import (
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestUnmarshalSpanContextConfig(t *testing.T) {
	original := trace.SpanContextConfig{
		TraceID:    mustTraceIDFromHex(t, "0123456789abcdef0123456789abcdef"),
		SpanID:     mustSpanIDFromHex(t, "0123456789abcdef"),
		TraceFlags: trace.FlagsSampled,
		TraceState: mustTraceState(t, "foo=bar,baz=qux"),
		Remote:     true,
	}

	sc := trace.NewSpanContext(original)

	data, err := sc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal span context: %v", err)
	}

	var decoded trace.SpanContextConfig
	if err := UnmarshalSpanContextConfig(data, &decoded); err != nil {
		t.Fatalf("unmarshal span context config: %v", err)
	}

	if decoded.TraceID != original.TraceID {
		t.Fatalf("trace id mismatch: got %s want %s", decoded.TraceID.String(), original.TraceID.String())
	}

	if decoded.SpanID != original.SpanID {
		t.Fatalf("span id mismatch: got %s want %s", decoded.SpanID.String(), original.SpanID.String())
	}

	if decoded.TraceFlags != original.TraceFlags {
		t.Fatalf("trace flags mismatch: got %s want %s", decoded.TraceFlags.String(), original.TraceFlags.String())
	}

	if decoded.TraceState.String() != original.TraceState.String() {
		t.Fatalf("trace state mismatch: got %q want %q", decoded.TraceState.String(), original.TraceState.String())
	}

	if decoded.Remote != original.Remote {
		t.Fatalf("remote mismatch: got %v want %v", decoded.Remote, original.Remote)
	}
}

func TestUnmarshalSpanContextConfig_Null(t *testing.T) {
	decoded := trace.SpanContextConfig{
		TraceID:    mustTraceIDFromHex(t, "0123456789abcdef0123456789abcdef"),
		SpanID:     mustSpanIDFromHex(t, "0123456789abcdef"),
		TraceFlags: trace.FlagsSampled,
		TraceState: mustTraceState(t, "foo=bar"),
		Remote:     true,
	}

	if err := UnmarshalSpanContextConfig([]byte("null"), &decoded); err != nil {
		t.Fatalf("unmarshal null span context config: %v", err)
	}

	if decoded.TraceID.IsValid() || decoded.SpanID.IsValid() || decoded.TraceFlags != 0 || decoded.TraceState.String() != "" || decoded.Remote {
		t.Fatalf("expected zero config, got %#v", decoded)
	}
}

func mustTraceIDFromHex(t *testing.T, value string) trace.TraceID {
	t.Helper()

	traceID, err := trace.TraceIDFromHex(value)
	if err != nil {
		t.Fatalf("parse trace id %q: %v", value, err)
	}

	return traceID
}

func mustSpanIDFromHex(t *testing.T, value string) trace.SpanID {
	t.Helper()

	spanID, err := trace.SpanIDFromHex(value)
	if err != nil {
		t.Fatalf("parse span id %q: %v", value, err)
	}

	return spanID
}

func mustTraceState(t *testing.T, value string) trace.TraceState {
	t.Helper()

	traceState, err := trace.ParseTraceState(value)
	if err != nil {
		t.Fatalf("parse trace state %q: %v", value, err)
	}

	return traceState
}
