package main

import (
	"context"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	bootstrap "github.com/LinPr/go-bootstrap"
	"github.com/LinPr/go-bootstrap/otel"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sdkotel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	// 配置 Prometheus 导出器
	config := &otel.Config{
		ServiceName:    "prometheus-example",
		ServiceVersion: "1.0.0",
		Log: otel.LogConfig{
			Enable: true,
			Type:   otel.ExporterTypeStdout,
			Pretty: false,
		},
		Trace: otel.TraceConfig{
			Enable:        true,
			Type:          otel.ExporterTypeStdout,
			Pretty:        false,
			SamplingRatio: 1.0,
		},
		Metric: otel.MetricConfig{
			Enable:               true,
			Type:                 otel.ExporterTypePrometheus,
			EnableRuntimeMetrics: true,
		},
	}

	if err := bootstrap.InitOtel(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bootstrap.ShutdownOtel(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	slog.Info("Prometheus example started")

	// 创建自定义指标
	meter := sdkotel.Meter("prometheus-example")

	// 计数器
	requestCounter, err := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}

	// 直方图
	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		log.Fatalf("Failed to create histogram: %v", err)
	}

	// 异步仪表
	_, err = meter.Int64ObservableGauge(
		"system.memory.usage",
		metric.WithDescription("Current memory usage"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			// 模拟内存使用情况
			observer.Observe(int64(rand.Intn(1000000000)))
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create gauge: %v", err)
	}

	// 启动 HTTP 服务器，暴露 Prometheus 指标
	http.Handle("/metrics", promhttp.Handler())

	// 模拟请求的处理器
	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		// 创建 trace
		tracer := sdkotel.Tracer("prometheus-example")
		_, span := tracer.Start(ctx, "handle-api-request")
		defer span.End()

		// 模拟处理
		processingTime := time.Duration(rand.Intn(200)) * time.Millisecond
		time.Sleep(processingTime)

		// 记录指标
		requestCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
				attribute.String("status", "200"),
			),
		)

		duration := float64(time.Since(start).Milliseconds())
		requestDuration.Record(ctx, duration,
			metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
			),
		)

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.path", r.URL.Path),
			attribute.Float64("http.duration_ms", duration),
		)

		slog.Info("Request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", duration,
		)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 启动后台任务模拟指标生成
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			requestCounter.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("method", "BACKGROUND"),
					attribute.String("path", "/background"),
					attribute.String("status", "200"),
				),
			)
		}
	}()

	slog.Info("Server starting", "address", ":8080")
	slog.Info("Prometheus metrics available at http://localhost:8080/metrics")
	slog.Info("Test API endpoint at http://localhost:8080/api/data")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
