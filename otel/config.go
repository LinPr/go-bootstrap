package otel

// ExporterType 定义导出器类型
type ExporterType string

const (
	ExporterTypeStdout ExporterType = "stdout"
	ExporterTypeHTTP   ExporterType = "http"
	ExporterTypeGRPC   ExporterType = "grpc"

	// ExporterTypePrometheus Prometheus 导出器（仅用于 Metric）
	ExporterTypePrometheus ExporterType = "prometheus"
)

// LoggerType 定义日志桥接类型
type LoggerType string

const (
	LoggerTypeSlog   LoggerType = "slog"   // 使用 otelslog 桥接
	LoggerTypeZap    LoggerType = "zap"    // 使用 otelzap 桥接
	LoggerTypeLogrus LoggerType = "logrus" // 使用 otellogrus 桥接
	LoggerTypeLogr   LoggerType = "logr"   // 使用 otellogr 桥接
)

// Config OpenTelemetry 初始化配置
type Config struct {
	// ServiceName 服务名称
	ServiceName string
	// ServiceVersion 服务版本
	ServiceVersion string
	// Log 日志配置
	Log LogConfig
	// Trace 追踪配置
	Trace TraceConfig
	// Metric 指标配置
	Metric MetricConfig
}

// LogConfig 日志配置
type LogConfig struct {
	// Enable 是否启用日志
	Enable bool
	// Exporter 导出器类型：stdout, http, grpc
	Exporter ExporterType
	// Logger 日志桥接类型：slog, zap, logrus, logr
	Logger LoggerType
	// RemoteAddr 远程地址（用于 http 和 grpc）
	RemoteAddr string
	// Headers 请求头（用于 http 和 grpc）
	Headers map[string]string
	// Pretty 是否美化输出（仅用于 stdout）
	Pretty bool
}

// TraceConfig 追踪配置
type TraceConfig struct {
	// Enable 是否启用追踪
	Enable bool
	// Exporter 导出器类型：stdout, http, grpc
	Exporter ExporterType
	// RemoteAddr 远程地址（用于 http 和 grpc）
	RemoteAddr string
	// Headers 请求头（用于 http 和 grpc）
	Headers map[string]string
	// Pretty 是否美化输出（仅用于 stdout）
	Pretty bool
	// SamplingRatio 采样率（0.0 到 1.0）
	SamplingRatio float64
}

// MetricConfig 指标配置
type MetricConfig struct {
	// Enable 是否启用指标
	Enable bool
	// Exporter 导出器类型：stdout, http, grpc, prometheus
	Exporter ExporterType
	// RemoteAddr 远程地址（用于 http 和 grpc）
	RemoteAddr string
	// Headers 请求头（用于 http 和 grpc）
	Headers map[string]string
	// Pretty 是否美化输出（仅用于 stdout）
	Pretty bool
	// IntervalSeconds 指标上报间隔（秒）
	IntervalSeconds int
	// EnableRuntimeMetrics 是否启用 Go Runtime 指标
	EnableRuntimeMetrics bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "default-service",
		ServiceVersion: "0.0.0",
		Log: LogConfig{
			Enable:   true,
			Exporter: ExporterTypeStdout,
			Logger:   LoggerTypeSlog,
			Pretty:   true,
		},
		Trace: TraceConfig{
			Enable:        true,
			Exporter:      ExporterTypeStdout,
			Pretty:        true,
			SamplingRatio: 1.0,
		},
		Metric: MetricConfig{
			Enable:               true,
			Exporter:             ExporterTypeStdout,
			Pretty:               true,
			IntervalSeconds:      10,
			EnableRuntimeMetrics: true,
		},
	}
}
