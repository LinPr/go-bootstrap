package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	bootstrap "github.com/LinPr/go-bootstrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	// 使用默认配置初始化
	config := bootstrap.DefaultConfig()
	config.ServiceName = "basic-example"
	config.ServiceVersion = "1.0.0"

	config.Metric.Enable = false
	config.Trace.Enable = false
	// 初始化 OpenTelemetry
	if err := bootstrap.InitOtel(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	// 确保在程序退出时关闭
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bootstrap.ShutdownOtel(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	// 使用日志
	slog.Info("Application started", "service", config.ServiceName)
	slog.Debug("Debug message", "key", "value")
	slog.Warn("Warning message", "warning_code", 123)

	// 使用追踪
	ctx := context.Background()
	tracer := otel.Tracer("basic-example")

	ctx, span := tracer.Start(ctx, "main-operation")
	span.SetAttributes(attribute.String("example", "basic"))
	defer span.End()

	// 模拟一些工作
	doWork(ctx)

	// 使用指标
	meter := otel.Meter("basic-example")
	counter, err := meter.Int64Counter("example.counter",
		metric.WithDescription("A simple counter"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create counter", "error", err)
		return
	}

	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("type", "basic")))
	counter.Add(ctx, 5, metric.WithAttributes(attribute.String("type", "batch")))

	slog.Info("Application finished successfully")

	// 等待一段时间以确保所有数据被导出
	time.Sleep(2 * time.Second)
}

func doWork(ctx context.Context) {
	tracer := otel.Tracer("basic-example")
	_, span := tracer.Start(ctx, "do-work")
	defer span.End()

	slog.Info("Doing some work...")
	time.Sleep(100 * time.Millisecond)

	span.SetAttributes(
		attribute.String("work.type", "example"),
		attribute.Int("work.duration_ms", 100),
	)
}
