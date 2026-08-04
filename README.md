# Go Bootstrap - OpenTelemetry 初始化库

[![Go Reference](https://pkg.go.dev/badge/github.com/LinPr/go-bootstrap.svg)](https://pkg.go.dev/github.com/LinPr/go-bootstrap)
[![Go Report Card](https://goreportcard.com/badge/github.com/LinPr/go-bootstrap)](https://goreportcard.com/report/github.com/LinPr/go-bootstrap)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一个简单易用的 Go 语言 OpenTelemetry SDK 初始化库，让你只需几行代码即可为应用添加完整的可观测性支持。

## 特性

- 🚀 **开箱即用** - 默认配置即可快速启动
- 📊 **全面支持** - 集成日志（Log）、追踪（Trace）、指标（Metric）
- 🔧 **灵活配置** - 支持多种导出器类型（Stdout、OTLP HTTP、OTLP gRPC、Prometheus）
- 🎯 **类型安全** - 使用强类型常量，避免字符串错误
- 🧪 **完整测试** - 包含单元测试和示例代码
- 📝 **结构化日志** - 自动集成 Go 标准库 `log/slog`
- 🌐 **分布式追踪** - 支持跨服务的上下文传播
- 📈 **Runtime 指标** - 可选启用 Go Runtime 性能指标

## 安装

```bash
go get github.com/LinPr/go-bootstrap
```

## 快速开始

### 使用默认配置

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "time"

    bootstrap "github.com/LinPr/go-bootstrap"
)

func main() {
    // 使用默认配置初始化
    config := bootstrap.DefaultConfig()
    config.ServiceName = "my-service"
    config.ServiceVersion = "1.0.0"

    if err := bootstrap.Initialize(config); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        bootstrap.Shutdown(ctx)
    }()

    // 现在可以使用标准的 slog 和 otel API
    slog.Info("Application started")
}
```

### 自定义配置

```go
config := &bootstrap.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Log: bootstrap.LogConfig{
        Enable:     true,
        Type:       bootstrap.ExporterTypeHTTP,
        RemoteAddr: "http://localhost:4318/v1/logs",
    },
    Trace: bootstrap.TraceConfig{
        Enable:        true,
        Type:          bootstrap.ExporterTypeGRPC,
        RemoteAddr:    "localhost:4317",
        SamplingRatio: 1.0,
    },
    Metric: bootstrap.MetricConfig{
        Enable:               true,
        Type:                 bootstrap.ExporterTypePrometheus,
        EnableRuntimeMetrics: true,
    },
}

if err := bootstrap.Initialize(config); err != nil {
    log.Fatalf("Failed to initialize: %v", err)
}
```

## 配置选项

### ExporterType 导出器类型

```go
const (
    ExporterTypeStdout     ExporterType = "stdout"      // 标准输出
    ExporterTypeHTTP       ExporterType = "http"        // OTLP HTTP
    ExporterTypeGRPC       ExporterType = "grpc"        // OTLP gRPC
    ExporterTypePrometheus ExporterType = "prometheus"  // Prometheus（仅 Metric）
)
```

### Config 配置结构

| 字段            | 类型           | 说明           |
|---------------|--------------|--------------|
| ServiceName   | string       | 服务名称         |
| ServiceVersion| string       | 服务版本         |
| Log           | LogConfig    | 日志配置         |
| Trace         | TraceConfig  | 追踪配置         |
| Metric        | MetricConfig | 指标配置         |

### LogConfig 日志配置

| 字段         | 类型              | 说明                      |
|------------|-----------------|-------------------------|
| Enable     | bool            | 是否启用日志                  |
| Type       | ExporterType    | 导出器类型                   |
| RemoteAddr | string          | 远程地址（用于 http 和 grpc）   |
| Headers    | map[string]string | 请求头（用于 http 和 grpc）    |
| Pretty     | bool            | 是否美化输出（仅用于 stdout）     |

### TraceConfig 追踪配置

| 字段            | 类型              | 说明                      |
|---------------|-----------------|-------------------------|
| Enable        | bool            | 是否启用追踪                  |
| Type          | ExporterType    | 导出器类型                   |
| RemoteAddr    | string          | 远程地址（用于 http 和 grpc）   |
| Headers       | map[string]string | 请求头（用于 http 和 grpc）    |
| Pretty        | bool            | 是否美化输出（仅用于 stdout）     |
| SamplingRatio | float64         | 采样率（0.0 到 1.0）          |

### MetricConfig 指标配置

| 字段                    | 类型              | 说明                          |
|-----------------------|-----------------|------------------------------|
| Enable                | bool            | 是否启用指标                      |
| Type                  | ExporterType    | 导出器类型                       |
| RemoteAddr            | string          | 远程地址（用于 http 和 grpc）       |
| Headers               | map[string]string | 请求头（用于 http 和 grpc）        |
| Pretty                | bool            | 是否美化输出（仅用于 stdout）         |
| IntervalSeconds       | int             | 指标上报间隔（秒）                   |
| EnableRuntimeMetrics  | bool            | 是否启用 Go Runtime 指标          |

## API 方法

### 初始化和关闭

```go
// Initialize 初始化全局 OpenTelemetry 提供者
func Initialize(config *Config) error

// Shutdown 关闭全局提供者
func Shutdown(ctx context.Context) error
```

### 创建自定义提供者

```go
// NewProvider 创建一个新的提供者实例
func NewProvider(config *Config) (*Provider, error)

// Shutdown 关闭提供者
func (p *Provider) Shutdown(ctx context.Context) error
```

### 获取提供者

```go
// GetGlobalProvider 获取全局提供者
func GetGlobalProvider() *Provider

// GetLoggerProvider 获取日志提供者
func GetLoggerProvider() *sdklog.LoggerProvider

// GetTracerProvider 获取追踪提供者
func GetTracerProvider() *sdktrace.TracerProvider

// GetMeterProvider 获取指标提供者
func GetMeterProvider() *sdkmetric.MeterProvider
```

## 使用示例

### 日志记录

```go
import "log/slog"

slog.Info("User logged in", "user_id", 12345, "ip", "192.168.1.1")
slog.Error("Database connection failed", "error", err)
```

### 分布式追踪

```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(ctx, "process-request")
defer span.End()

span.SetAttributes(
    attribute.String("request.id", "req-123"),
    attribute.Int("user.id", 12345),
)
```

### 指标收集

```go
import "go.opentelemetry.io/otel"

meter := otel.Meter("my-service")
counter, _ := meter.Int64Counter("http.requests")
counter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("method", "GET"),
    attribute.String("path", "/api/users"),
))
```

## 完整示例

项目包含多个完整示例，位于 `examples/` 目录：

- [basic](examples/basic) - 基础使用示例
- [otlp-http](examples/otlp-http) - OTLP HTTP 导出器示例
- [otlp-grpc](examples/otlp-grpc) - OTLP gRPC 导出器示例
- [prometheus](examples/prometheus) - Prometheus 导出器示例
- [custom](examples/custom) - 自定义配置示例
- [advanced](examples/advanced) - 高级用法示例（嵌套 span、错误处理）

### 运行示例

```bash
# 基础示例
cd examples/basic
go run main.go

# Prometheus 示例（需要访问 http://localhost:8080/metrics）
cd examples/prometheus
go run main.go
```

## 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 与 OpenTelemetry Collector 集成

### 使用 Docker 启动 Collector

```yaml
# docker-compose.yml
version: '3'
services:
  otel-collector:
    image: otel/opentelemetry-collector:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
```

```bash
docker-compose up -d
```

### 应用配置

```go
config := &bootstrap.Config{
    ServiceName: "my-service",
    Log: bootstrap.LogConfig{
        Enable:     true,
        Type:       bootstrap.ExporterTypeGRPC,
        RemoteAddr: "localhost:4317",
    },
    Trace: bootstrap.TraceConfig{
        Enable:     true,
        Type:       bootstrap.ExporterTypeGRPC,
        RemoteAddr: "localhost:4317",
    },
    Metric: bootstrap.MetricConfig{
        Enable:     true,
        Type:       bootstrap.ExporterTypeGRPC,
        RemoteAddr: "localhost:4317",
    },
}
```

## 最佳实践

1. **始终调用 Shutdown** - 确保在应用退出时正确关闭提供者，以便刷新所有待处理的数据
2. **使用上下文传播** - 在函数之间传递 `context.Context` 以保持追踪链路
3. **合理设置采样率** - 生产环境可以降低采样率以减少开销
4. **添加有意义的属性** - 为 span 和 metric 添加业务相关的属性
5. **处理错误** - 使用 `span.RecordError()` 记录错误信息
6. **使用结构化日志** - 使用 `slog` 的键值对而不是字符串拼接

## 环境变量

库支持通过环境变量配置资源属性：

```bash
export OTEL_SERVICE_NAME=my-service
export OTEL_SERVICE_VERSION=1.0.0
export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production,team=backend
```

## 依赖

- Go 1.21+
- go.opentelemetry.io/otel v1.30.0+
- go.opentelemetry.io/contrib v1.30.0+

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0 (2024-08-04)

- 🎉 首次发布
- ✨ 支持 Log、Trace、Metric 初始化
- 🔧 支持多种导出器类型
- 📝 完整的文档和示例
- 🧪 单元测试覆盖

## 相关链接

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)

## 支持

如有问题或建议，请通过以下方式联系：

- 提交 [GitHub Issue](https://github.com/LinPr/go-bootstrap/issues)
- 发送邮件至 your.email@example.com