package zapsugar

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapSugar wraps zap.SugaredLogger with context-aware logging methods.
type ZapSugar struct {
	logger *zap.SugaredLogger
}

// NewSubScopedZapSugar creates a new ZapSugar logger with a named scope.
func NewSubScopedZapSugar(name string, logger *zap.SugaredLogger) *ZapSugar {
	// If logger is nil, use the global zap.S() logger and create a new sub-scoped logger with the given name.
	if logger == nil {
		return &ZapSugar{
			logger: zap.S().Named(name),
		}
	}

	// Otherwise, create a new sub-scoped logger from the provided logger.
	return &ZapSugar{
		logger: logger.Named(name),
	}
}

func (s *ZapSugar) Logger() *zap.SugaredLogger {
	return s.logger
}

type baggageCoreWrapper struct {
	zapcore.Core
	memberSet map[string]struct{}
}

func newBaggageCoreWrapper(core zapcore.Core, baggageMembers map[string]struct{}) *baggageCoreWrapper {
	return &baggageCoreWrapper{
		Core:      core,
		memberSet: baggageMembers,
	}
}

func (b *baggageCoreWrapper) With(fields []zapcore.Field) zapcore.Core {
	fields = b.extractBaggageMembers(fields)

	return &baggageCoreWrapper{
		Core:      b.Core.With(fields),
		memberSet: b.memberSet,
	}
}

func (b *baggageCoreWrapper) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	fields = b.extractBaggageMembers(fields)
	return b.Core.Write(entry, fields)
}

func (b *baggageCoreWrapper) extractBaggageMembers(fields []zapcore.Field) []zapcore.Field {
	if len(fields) > 0 {
		for _, field := range fields {
			if ctxFld, ok := field.Interface.(context.Context); ok {
				baggageMembers := baggage.FromContext(ctxFld).Members()
				for _, member := range baggageMembers {
					if _, ok := b.memberSet[member.Key()]; ok {
						fields = append(fields, zap.String(member.Key(), member.Value()))
					}
				}
				break
			}
		}
	}
	return fields
}

func (s *ZapSugar) WithBaggageMembers(members ...string) *ZapSugar {
	logger := s.logger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			memberSet := make(map[string]struct{})
			for _, member := range members {
				memberSet[member] = struct{}{}
			}
			return newBaggageCoreWrapper(core, memberSet)
		}),
	)
	s.logger = logger
	return s
}

// WithOptions applies zap options to the logger.
func (s *ZapSugar) WithOptions(opts ...zap.Option) *ZapSugar {
	s.logger = s.logger.WithOptions(opts...)
	return s
}

// WithAttribute adds a key-value attribute to the logger.
func (s *ZapSugar) WithAttribute(name string, args any) *ZapSugar {
	s.logger = s.logger.With(name, args)
	return s
}

// Debugw logs a debug message with context and key-value pairs.
func (s *ZapSugar) Debugw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Debugw(msg, args...)
}

// Infow logs an info message with context and key-value pairs.
func (s *ZapSugar) Infow(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Infow(msg, args...)
}

// Warnw logs a warn message with context and key-value pairs.
func (s *ZapSugar) Warnw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Warnw(msg, args...)
}

// Errorw logs an error message with context and key-value pairs.
func (s *ZapSugar) Errorw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Errorw(msg, args...)
}

// Debugf logs a formatted debug message with context.
func (s *ZapSugar) Debugf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Debugf(template, args...)
}

// Infof logs a formatted info message with context.
func (s *ZapSugar) Infof(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Infof(template, args...)
}

// Warnf logs a formatted warn message with context.
func (s *ZapSugar) Warnf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Warnf(template, args...)
}

// Errorf logs a formatted error message with context.
func (s *ZapSugar) Errorf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Errorf(template, args...)
}

func Debugw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Debugw(msg, args...)
}

func Infow(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Infow(msg, args...)
}

func Warnw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Warnw(msg, args...)
}

func Errorw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Errorw(msg, args...)
}

// Debugf logs a formatted debug message with context.
func Debugf(ctx context.Context, template string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Debugf(template, args...)
}

// Infof logs a formatted info message with context.
func Infof(ctx context.Context, template string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Infof(template, args...)
}

// Warnf logs a formatted warn message with context.
func Warnf(ctx context.Context, template string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Warnf(template, args...)
}

// Errorf logs a formatted error message with context.
func Errorf(ctx context.Context, template string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Errorf(template, args...)
}
