package otel

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

// otelLogProvider embeds otellog.LoggerProvider and adds a global severity floor.
type otelLogProvider struct {
	otellog.LoggerProvider
	minSeverity otellog.Severity
}

// otelLogProvider wraps the underlying OTEL logger provider so we can inject
// a minimum severity filter before slog emits records.
func (p otelLogProvider) Logger(name string, options ...otellog.LoggerOption) otellog.Logger {
	return otelSeverityLogger{
		Logger:      p.LoggerProvider.Logger(name, options...),
		minSeverity: p.minSeverity,
	}
}

// otelSeverityLogger forwards to the real OTEL logger, but rejects records
// below the configured severity before the bridge gets a chance to emit them.
type otelSeverityLogger struct {
	otellog.Logger
	minSeverity otellog.Severity
}

// Enabled applies the configured minimum severity and then delegates to the
// wrapped logger for the provider's own filtering and lifecycle checks.
func (l otelSeverityLogger) Enabled(ctx context.Context, param otellog.EnabledParameters) bool {
	if param.Severity < l.minSeverity {
		return false
	}
	return l.Logger.Enabled(ctx, param)
}

func parseLogLevel(level string) (otellog.Severity, zapcore.Level, error) {
	// Normalize the level string to lowercase and trim whitespace for comparison.
	normalized := cmp.Or(strings.ToLower(strings.TrimSpace(level)), "info")

	switch normalized {
	case "debug":
		return otellog.SeverityDebug, zapcore.DebugLevel, nil
	case "info":
		return otellog.SeverityInfo, zapcore.InfoLevel, nil
	case "warn":
		return otellog.SeverityWarn, zapcore.WarnLevel, nil
	case "error":
		return otellog.SeverityError, zapcore.ErrorLevel, nil
	default:
		return 0, 0, fmt.Errorf("unsupported log level: %s", level)
	}
}
