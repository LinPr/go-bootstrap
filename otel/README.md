# go-bootstrap/otel

OpenTelemetry 初始化工具（Log / Trace / Metric）。

## 安装

```bash
go get github.com/LinPr/go-bootstrap/otel
```

## 快速开始

```go
package main

import (
	"context"
	"log"

	bsotel "github.com/LinPr/go-bootstrap/otel"
)

func main() {
	cfg := bsotel.DefaultConfig()
	cfg.ServiceName = "my-service"
	cfg.ServiceVersion = "1.0.0"

	if err := bsotel.InitOtelProvider(cfg); err != nil {
		log.Fatal(err)
	}
	defer bsotel.ShutdownOtelProvider(context.Background())
}
```

## 配置示例

```go
cfg := &bsotel.Config{
	ServiceName:    "my-service",
	ServiceVersion: "1.0.0",
	Log: bsotel.LogConfig{
		Enable:     true,
		Exporter:   bsotel.ExporterTypeHTTP,
		Logger:     bsotel.LoggerTypeSlog,
		RemoteAddr: "http://localhost:4318/v1/logs",
	},
	Trace: bsotel.TraceConfig{
		Enable:        true,
		Exporter:      bsotel.ExporterTypeGRPC,
		RemoteAddr:    "localhost:4317",
		SamplingRatio: 1.0,
	},
	Metric: bsotel.MetricConfig{
		Enable:               true,
		Exporter:             bsotel.ExporterTypePrometheus,
		EnableRuntimeMetrics: true,
	},
}
```

## 常用常量

- 导出器：
  - `ExporterTypeStdout`
  - `ExporterTypeHTTP`
  - `ExporterTypeGRPC`
  - `ExporterTypePrometheus`（仅 Metric）
- 日志桥接：
  - `LoggerTypeSlog`
  - `LoggerTypeZap`
  - `LoggerTypeLogrus`
  - `LoggerTypeLogr`

## 测试

```bash
cd otel
go test ./...
```
