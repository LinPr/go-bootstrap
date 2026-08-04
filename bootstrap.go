package gobootstrap

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var globalProvider *Provider

// Initialize 使用提供的配置初始化 OpenTelemetry SDK
// 这是一个便捷方法，会设置全局提供者
func Initialize(config *Config) error {
	provider, err := NewProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	globalProvider = provider
	return nil
}

// Shutdown 关闭全局 OpenTelemetry 提供者
func Shutdown(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}

// GetGlobalProvider 获取全局提供者
func GetGlobalProvider() *Provider {
	return globalProvider
}

// GetLoggerProvider 获取全局日志提供者
func GetLoggerProvider() *sdklog.LoggerProvider {
	if globalProvider != nil {
		return globalProvider.GetLoggerProvider()
	}
	return global.GetLoggerProvider().(*sdklog.LoggerProvider)
}

// GetTracerProvider 获取全局追踪提供者
func GetTracerProvider() *sdktrace.TracerProvider {
	if globalProvider != nil {
		return globalProvider.GetTracerProvider()
	}
	return otel.GetTracerProvider().(*sdktrace.TracerProvider)
}

// GetMeterProvider 获取全局指标提供者
func GetMeterProvider() *sdkmetric.MeterProvider {
	if globalProvider != nil {
		return globalProvider.GetMeterProvider()
	}
	return otel.GetMeterProvider().(*sdkmetric.MeterProvider)
}
