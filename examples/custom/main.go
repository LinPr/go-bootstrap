package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	bsotel "github.com/LinPr/go-bootstrap/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	// 自定义配置 - 只启用部分功能
	config := &bsotel.Config{
		ServiceName:    "custom-example",
		ServiceVersion: "2.1.0",
		// 只启用日志，禁用追踪和指标
		Log: bsotel.LogConfig{
			Enable: true,
			Type:   bsotel.ExporterTypeStdout,
			Pretty: true,
		},
		Trace: bsotel.TraceConfig{
			Enable: false, // 禁用追踪
		},
		Metric: bsotel.MetricConfig{
			Enable: false, // 禁用指标
		},
	}

	if err := bsotel.InitOtelProvider(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bsotel.ShutdownOtelProvider(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	slog.Info("Custom configuration example started",
		"service", config.ServiceName,
		"version", config.ServiceVersion,
	)

	// 即使追踪被禁用，代码仍然可以调用，但不会产生任何输出
	ctx := context.Background()
	tracer := otel.Tracer("custom-example")
	_, span := tracer.Start(ctx, "disabled-trace-operation")
	span.SetAttributes(attribute.String("note", "this trace is disabled"))
	span.End()

	// 日志会正常工作
	slog.Debug("Debug log message")
	slog.Info("Info log message", "key1", "value1", "key2", 123)
	slog.Warn("Warning log message", "warning_code", "WARN001")
	slog.Error("Error log message", "error_code", "ERR001", "details", "Something went wrong")

	// 使用结构化日志
	slog.With(
		"component", "database",
		"operation", "query",
	).Info("Database query executed",
		"query_time_ms", 45,
		"rows_affected", 10,
	)

	slog.Info("Custom configuration example completed")

	time.Sleep(2 * time.Second)
}
