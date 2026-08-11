package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var globalProvider *OtelProviders

// NewOtelProviders 使用提供的配置初始化 OpenTelemetry SDK。
// 这是一个便捷方法，会设置并返回全局提供者。
func NewOtelProviders(config *Config) (*OtelProviders, error) {
	providers, err := newOtelProviders(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	globalProvider = providers
	return providers, nil
}

// Shutdown 关闭全局 OpenTelemetry 提供者
func ShutdownOtelProvider(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}

// GetOtelProvider 获取全局提供者。
func GetOtelProvider() *OtelProviders {
	return globalProvider
}

// GetLoggerProvider 获取日志提供者
func (p *OtelProviders) GetLoggerProvider() log.LoggerProvider {
	return p.logProvider
}

// GetTracerProvider 获取追踪提供者
func (p *OtelProviders) GetTracerProvider() trace.TracerProvider {
	return p.traceProvider
}

// GetMeterProvider 获取指标提供者
func (p *OtelProviders) GetMeterProvider() metric.MeterProvider {
	return p.metricProvider
}
