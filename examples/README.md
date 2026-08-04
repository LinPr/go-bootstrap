# Go Bootstrap Examples

这个目录包含了 `go-bootstrap` 库的多个使用示例。

## 示例列表

### 1. basic - 基础使用
最简单的使用示例，展示如何使用默认配置快速开始。

```bash
cd basic
go run main.go
```

**特点：**
- 使用 `DefaultConfig()` 默认配置
- 演示日志、追踪、指标的基本用法
- 适合快速入门

### 2. otlp-http - OTLP HTTP 导出器
展示如何配置 OTLP HTTP 协议导出器，将数据发送到远程收集器。

```bash
cd otlp-http
go run main.go
```

**特点：**
- HTTP 协议导出
- 自定义请求头
- 适用于 OpenTelemetry Collector

**前置条件：**
需要运行 OTLP 接收器（如 OpenTelemetry Collector）在 `localhost:4318`

### 3. otlp-grpc - OTLP gRPC 导出器
展示如何配置 OTLP gRPC 协议导出器。

```bash
cd otlp-grpc
go run main.go
```

**特点：**
- gRPC 协议导出
- 支持认证头
- 高性能传输

**前置条件：**
需要运行 OTLP 接收器（如 OpenTelemetry Collector）在 `localhost:4317`

### 4. prometheus - Prometheus 导出器
展示如何使用 Prometheus 导出器，通过 HTTP 端点暴露指标。

```bash
cd prometheus
go run main.go
```

然后访问：
- 应用 API: http://localhost:8080/api/data
- Prometheus 指标: http://localhost:8080/metrics

**特点：**
- Prometheus 拉取模式
- 自定义指标（Counter、Histogram、Gauge）
- HTTP 服务器集成
- Go Runtime 指标

### 5. custom - 自定义配置
展示如何根据需求自定义配置，选择性启用功能。

```bash
cd custom
go run main.go
```

**特点：**
- 选择性启用 Log/Trace/Metric
- 展示配置的灵活性
- 适合生产环境按需配置

### 6. advanced - 高级用法
展示复杂场景下的使用方法，包括嵌套 span、错误处理等。

```bash
cd advanced
go run main.go
```

**特点：**
- 嵌套的 span（父子关系）
- 错误记录和状态管理
- 多级函数调用追踪
- 业务属性添加
- 模拟真实业务场景

## 运行所有示例

```bash
#!/bin/bash
for dir in basic otlp-http otlp-grpc prometheus custom advanced; do
    echo "Running $dir example..."
    cd $dir
    go run main.go
    cd ..
    echo "---"
done
```

## 通用依赖

所有示例都需要安装依赖：

```bash
# 在项目根目录运行
go mod download
go mod tidy
```

## 使用 OpenTelemetry Collector

如果要运行 OTLP 示例，建议使用 Docker 启动 Collector：

```bash
# 创建配置文件 otel-collector-config.yaml
cat > otel-collector-config.yaml <<EOF
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  logging:
    loglevel: debug

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [logging]
    metrics:
      receivers: [otlp]
      exporters: [logging]
    logs:
      receivers: [otlp]
      exporters: [logging]
EOF

# 启动 Collector
docker run -d \
  --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

查看 Collector 日志：
```bash
docker logs -f otel-collector
```

停止 Collector：
```bash
docker stop otel-collector
docker rm otel-collector
```

## 常见问题

### Q: 示例无法连接到 OTLP Collector
A: 确保 Collector 正在运行并监听正确的端口（4317 for gRPC, 4318 for HTTP）

### Q: Prometheus 示例看不到指标
A: 等待几秒钟让指标生成，然后访问 http://localhost:8080/metrics

### Q: 如何修改示例配置？
A: 直接编辑示例中的 `config` 变量，修改导出器类型、地址等参数

## 学习路径建议

1. **初学者**: basic → custom → prometheus
2. **集成 Collector**: otlp-http → otlp-grpc
3. **生产使用**: advanced → custom + otlp-grpc

## 更多资源

- [项目 README](../README.md)
- [OpenTelemetry 文档](https://opentelemetry.io/docs/)
- [Go SDK 文档](https://pkg.go.dev/go.opentelemetry.io/otel)
