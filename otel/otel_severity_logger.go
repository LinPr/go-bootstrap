package otel

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

// otelLogProvider embeds otellog.LoggerProvider and adds a global severity floor.
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

func parseLogLevel(level string) (otellog.Severity, zapcore.Level, logrus.Level, error) {
	// Normalize the level string to lowercase and trim whitespace for comparison.
	normalized := strings.ToLower(strings.TrimSpace(level))
	if normalized == "" {
		normalized = "info"
	}

	switch normalized {
	case "debug":
		return otellog.SeverityDebug, zapcore.DebugLevel, logrus.DebugLevel, nil
	case "info":
		return otellog.SeverityInfo, zapcore.InfoLevel, logrus.InfoLevel, nil
	case "warn":
		return otellog.SeverityWarn, zapcore.WarnLevel, logrus.WarnLevel, nil
	case "error":
		return otellog.SeverityError, zapcore.ErrorLevel, logrus.ErrorLevel, nil
	default:
		return 0, 0, 0, fmt.Errorf("unsupported log level: %s", level)
	}
}
