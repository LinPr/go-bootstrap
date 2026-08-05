package gobootstrap

import (
	"context"

	bsotel "github.com/LinPr/go-bootstrap/otel"
)

// InitOtel 初始化 OpenTelemetry SDK
func InitOtel(config *bsotel.Config) error {
	return bsotel.InitOtelProvider(config)
}

// ShutdownOtel 关闭 OpenTelemetry SDK
func ShutdownOtel(ctx context.Context) error {
	return bsotel.ShutdownOtelProvider(ctx)
}

// GetOtelProvider 获取全局 Provider
func GetOtelProvider() *bsotel.Provider {
	return bsotel.GetOtelProvider()
}

// DefaultConfig 返回默认配置
func DefaultConfig() *bsotel.Config {
	return bsotel.DefaultConfig()
}
