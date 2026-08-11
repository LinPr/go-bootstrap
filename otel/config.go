package otel

// ExporterType defines exporter types.
type ExporterType string

const (
	ExporterTypeStdout ExporterType = "stdout"
	ExporterTypeHTTP   ExporterType = "http"
	ExporterTypeGRPC   ExporterType = "grpc"

	// ExporterTypePrometheus is the Prometheus exporter (Metric only).
	ExporterTypePrometheus ExporterType = "prometheus"
)

// LoggerType defines log bridge types.
type LoggerType string

const (
	LoggerTypeSlog   LoggerType = "slog"   // Uses the otelslog bridge.
	LoggerTypeZap    LoggerType = "zap"    // Uses the otelzap bridge.
	LoggerTypeLogrus LoggerType = "logrus" // Uses the otellogrus bridge.
	LoggerTypeLogr   LoggerType = "logr"   // Uses the otellogr bridge.
)

// Config holds the OpenTelemetry initialization config.
type Config struct {
	// ServiceName is the service name.
	ServiceName string
	// ServiceVersion is the service version.
	ServiceVersion string
	// Log is the log configuration.
	Log LogConfig
	// Trace is the trace configuration.
	Trace TraceConfig
	// Metric is the metric configuration.
	Metric MetricConfig
}

// LogConfig holds log settings.
type LogConfig struct {
	// Enable toggles logging.
	Enable bool
	// Exporter is the exporter type: stdout, http, grpc.
	Exporter ExporterType
	// Logger is the log bridge type: slog, zap, logrus, logr.
	Logger LoggerType
	// Level is the log level: debug, info, warn, error.
	Level string
	// RemoteAddr is the remote address for HTTP and gRPC.
	RemoteAddr string
	// Headers contains request headers for HTTP and gRPC.
	Headers map[string]string
	// Pretty enables pretty output for stdout only.
	Pretty bool
}

// TraceConfig holds trace settings.
type TraceConfig struct {
	// Enable toggles tracing.
	Enable bool
	// Exporter is the exporter type: stdout, http, grpc.
	Exporter ExporterType
	// RemoteAddr is the remote address for HTTP and gRPC.
	RemoteAddr string
	// Headers contains request headers for HTTP and gRPC.
	Headers map[string]string
	// Pretty enables pretty output for stdout only.
	Pretty bool
	// SamplingRatio is the sampling ratio, from 0.0 to 1.0.
	SamplingRatio float64
}

// MetricConfig holds metric settings.
type MetricConfig struct {
	// Enable toggles metrics.
	Enable bool
	// Exporter is the exporter type: stdout, http, grpc, prometheus.
	Exporter ExporterType
	// RemoteAddr is the remote address for HTTP and gRPC.
	RemoteAddr string
	// Headers contains request headers for HTTP and gRPC.
	Headers map[string]string
	// Pretty enables pretty output for stdout only.
	Pretty bool
	// IntervalSeconds is the metric export interval in seconds.
	IntervalSeconds int
	// EnableRuntimeMetrics toggles Go runtime metrics.
	EnableRuntimeMetrics bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "default-service",
		ServiceVersion: "0.0.0",
		Log: LogConfig{
			Enable:   true,
			Exporter: ExporterTypeStdout,
			Logger:   LoggerTypeSlog,
			Level:    "info",
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
