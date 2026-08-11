package otel

import (
	"context"

	otellog "go.opentelemetry.io/otel/log"
)

// 继承 otellog.LoggerProvider 并添加全局日志级别
type otelLogProvider struct {
	otellog.LoggerProvider
	MinSeverity otellog.Severity
}

// otelLogProvider wraps the underlying OTEL logger provider so we can inject
// a minimum severity filter before slog emits records.
func (p otelLogProvider) Logger(name string, options ...otellog.LoggerOption) otellog.Logger {
	return otelSeverityLogger{
		Logger:      p.LoggerProvider.Logger(name, options...),
		MinSeverity: p.MinSeverity,
	}
}

// otelSeverityLogger forwards to the real OTEL logger, but rejects records
// below the configured severity before the bridge gets a chance to emit them.
type otelSeverityLogger struct {
	otellog.Logger
	MinSeverity otellog.Severity
}

// Enabled applies the configured minimum severity and then delegates to the
// wrapped logger for the provider's own filtering and lifecycle checks.
func (l otelSeverityLogger) Enabled(ctx context.Context, param otellog.EnabledParameters) bool {
	if param.Severity < l.MinSeverity {
		return false
	}
	return l.Logger.Enabled(ctx, param)
}
