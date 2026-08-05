package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	bsotel "github.com/LinPr/go-bootstrap"
	"github.com/LinPr/go-bootstrap/otel"
	sdkotel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	// 配置 OTLP gRPC 导出器
	config := &otel.Config{
		ServiceName:    "otlp-grpc-example",
		ServiceVersion: "1.0.0",
		Log: otel.LogConfig{
			Enable:     true,
			Type:       otel.ExporterTypeGRPC,
			RemoteAddr: "localhost:4317",
			Headers: map[string]string{
				"authorization": "Bearer your-token-here",
			},
		},
		Trace: otel.TraceConfig{
			Enable:        true,
			Type:          otel.ExporterTypeGRPC,
			RemoteAddr:    "localhost:4317",
			SamplingRatio: 1.0,
			Headers: map[string]string{
				"authorization": "Bearer your-token-here",
			},
		},
		Metric: otel.MetricConfig{
			Enable:               true,
			Type:                 otel.ExporterTypeGRPC,
			RemoteAddr:           "localhost:4317",
			IntervalSeconds:      10,
			EnableRuntimeMetrics: true,
			Headers: map[string]string{
				"authorization": "Bearer your-token-here",
			},
		},
	}

	if err := bsotel.InitOtel(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bsotel.ShutdownOtel(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	slog.Info("OTLP gRPC example started")

	// 创建追踪
	ctx := context.Background()
	tracer := sdkotel.Tracer("otlp-grpc-example")

	ctx, span := tracer.Start(ctx, "grpc-export-operation")
	span.SetAttributes(
		attribute.String("exporter.type", "grpc"),
		attribute.String("protocol", "otlp"),
	)

	// 模拟业务逻辑
	performTask(ctx)

	span.End()

	slog.Info("OTLP gRPC example completed")

	// 等待数据导出
	time.Sleep(3 * time.Second)
}

func performTask(ctx context.Context) {
	tracer := sdkotel.Tracer("otlp-grpc-example")
	_, span := tracer.Start(ctx, "perform-task")
	defer span.End()

	slog.Info("Performing task", "task_id", "task-67890")

	// 模拟处理时间
	time.Sleep(150 * time.Millisecond)

	span.SetAttributes(
		attribute.String("task.id", "task-67890"),
		attribute.String("task.type", "background"),
		attribute.String("task.status", "completed"),
	)

	slog.Info("Task completed successfully")
}
