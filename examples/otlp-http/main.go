package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	bootstrap "github.com/LinPr/go-bootstrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	// 配置 OTLP HTTP 导出器
	config := &bootstrap.Config{
		ServiceName:    "otlp-http-example",
		ServiceVersion: "1.0.0",
		Log: bootstrap.LogConfig{
			Enable:     true,
			Type:       bootstrap.ExporterTypeHTTP,
			RemoteAddr: "http://localhost:4318/v1/logs",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
		Trace: bootstrap.TraceConfig{
			Enable:        true,
			Type:          bootstrap.ExporterTypeHTTP,
			RemoteAddr:    "http://localhost:4318/v1/traces",
			SamplingRatio: 1.0,
		},
		Metric: bootstrap.MetricConfig{
			Enable:               true,
			Type:                 bootstrap.ExporterTypeHTTP,
			RemoteAddr:           "http://localhost:4318/v1/metrics",
			IntervalSeconds:      10,
			EnableRuntimeMetrics: true,
		},
	}

	if err := bootstrap.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bootstrap.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	slog.Info("OTLP HTTP example started")

	// 创建追踪
	ctx := context.Background()
	tracer := otel.Tracer("otlp-http-example")

	ctx, span := tracer.Start(ctx, "http-export-operation")
	span.SetAttributes(
		attribute.String("exporter.type", "http"),
		attribute.String("protocol", "otlp"),
	)

	// 模拟业务逻辑
	processRequest(ctx)

	span.End()

	slog.Info("OTLP HTTP example completed")

	// 等待数据导出
	time.Sleep(3 * time.Second)
}

func processRequest(ctx context.Context) {
	tracer := otel.Tracer("otlp-http-example")
	_, span := tracer.Start(ctx, "process-request")
	defer span.End()

	slog.Info("Processing request", "request_id", "req-12345")

	// 模拟处理时间
	time.Sleep(200 * time.Millisecond)

	span.SetAttributes(
		attribute.String("request.id", "req-12345"),
		attribute.Int("request.size", 1024),
		attribute.String("request.status", "success"),
	)

	slog.Info("Request processed successfully")
}
