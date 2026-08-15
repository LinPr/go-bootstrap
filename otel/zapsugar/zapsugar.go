package zapsugar

import (
	"context"

	"go.uber.org/zap"
)

// ZapSUgar wraps zap.SugaredLogger with context-aware logging methods.
type ZapSUgar struct {
	logger *zap.SugaredLogger
}

// NewSubScopedZapSugar creates a new ZapSUgar logger with a named scope.
func NewSubScopedZapSugar(name string, logger *ZapSUgar) *ZapSUgar {
	// If logger is nil, use the global zap.S() logger and create a new sub-scoped logger with the given name.
	if logger == nil {
		return &ZapSUgar{
			logger: zap.S().Named(name),
		}
	}

	// Otherwise, create a new sub-scoped logger from the provided logger.
	return &ZapSUgar{
		logger: logger.logger.Named(name),
	}
}

// WithOptions applies zap options to the logger.
func (s *ZapSUgar) WithOptions(opts ...zap.Option) *ZapSUgar {
	s.logger = s.logger.WithOptions(opts...)
	return s
}

// WithAttribute adds a key-value attribute to the logger.
func (s *ZapSUgar) WithAttribute(name string, args any) *ZapSUgar {
	s.logger = s.logger.With(name, args)
	return s
}

// Debugw logs a debug message with context and key-value pairs.
func (s *ZapSUgar) Debugw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).Debugw(msg, append([]any{"context", ctx}, args...)...)
}

// Infow logs an info message with context and key-value pairs.
func (s *ZapSUgar) Infow(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).Infow(msg, append([]any{"context", ctx}, args...)...)
}

// Warnw logs a warn message with context and key-value pairs.
func (s *ZapSUgar) Warnw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).Warnw(msg, append([]any{"context", ctx}, args...)...)
}

// Errorw logs an error message with context and key-value pairs.
func (s *ZapSUgar) Errorw(ctx context.Context, msg string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).Errorw(msg, append([]any{"context", ctx}, args...)...)
}

// Debugf logs a formatted debug message with context.
func (s *ZapSUgar) Debugf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Debugf(template, args...)
}

// Infof logs a formatted info message with context.
func (s *ZapSUgar) Infof(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Infof(template, args...)
}

// Warnf logs a formatted warn message with context.
func (s *ZapSUgar) Warnf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Warnf(template, args...)
}

// Errorf logs a formatted error message with context.
func (s *ZapSUgar) Errorf(ctx context.Context, template string, args ...any) {
	s.logger.WithOptions(zap.AddCallerSkip(1)).With("context", ctx).Errorf(template, args...)
}

// Debugw logs a debug message with context and key-value pairs.
func Debugw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).Debugw(msg, append([]any{"context", ctx}, args...)...)
}

// Infow logs an info message with context and key-value pairs.
func Infow(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).Infow(msg, append([]any{"context", ctx}, args...)...)
}

// Warnw logs a warn message with context and key-value pairs.
func Warnw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).Warnw(msg, append([]any{"context", ctx}, args...)...)
}

// Errorw logs an error message with context and key-value pairs.
func Errorw(ctx context.Context, msg string, args ...any) {
	zap.S().WithOptions(zap.AddCallerSkip(1)).Errorw(msg, append([]any{"context", ctx}, args...)...)
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
