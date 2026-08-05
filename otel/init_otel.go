package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var globalProvider *Provider

// Initialize 使用提供的配置初始化 OpenTelemetry SDK
// 这是一个便捷方法，会设置全局提供者
func InitOtelProvider(config *Config) error {
	provider, err := newProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	globalProvider = provider
	return nil
}

// Shutdown 关闭全局 OpenTelemetry 提供者
func ShutdownOtelProvider(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}

// GetGlobalProvider 获取全局提供者
func GetOtelProvider() *Provider {
	return globalProvider
}

// GetLoggerProvider 获取日志提供者
func (p *Provider) GetLoggerProvider() log.LoggerProvider {
	return p.logProvider
}

// GetTracerProvider 获取追踪提供者
func (p *Provider) GetTracerProvider() trace.TracerProvider {
	return p.traceProvider
}

// GetMeterProvider 获取指标提供者
func (p *Provider) GetMeterProvider() metric.MeterProvider {
	return p.metricProvider
}
