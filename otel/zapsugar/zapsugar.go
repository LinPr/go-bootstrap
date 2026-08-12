package zapsugar

import (
	"context"

	"go.uber.org/zap"
)

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
