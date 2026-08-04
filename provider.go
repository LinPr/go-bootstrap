package gobootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// Provider OpenTelemetry 提供者
type Provider struct {
	logProvider    *log.LoggerProvider
	traceProvider  *trace.TracerProvider
	metricProvider *metric.MeterProvider
}

// NewProvider 创建一个新的 OpenTelemetry 提供者
func NewProvider(config *Config) (*Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	p := &Provider{
		logProvider:    nil,
		traceProvider:  nil,
		metricProvider: nil,
	}

	if err := p.initialize(config); err != nil {
		return nil, err
	}

	return p, nil
}

// initialize 初始化 OpenTelemetry SDK
func (p *Provider) initialize(c *Config) error {
	// 设置上下文传播器
	p.initPropagator()

	// 创建资源
	res, err := p.createResource(c.ServiceName, c.ServiceVersion)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// 初始化 Log
	if err := p.initLog(&c.Log, res); err != nil {
		return fmt.Errorf("failed to initialize log provider: %w", err)
	}

	// 初始化 Trace
	if err := p.initTrace(c.Trace, res); err != nil {
		return fmt.Errorf("failed to initialize trace provider: %w", err)
	}

	// 初始化 Metric
	if err := p.initMetric(c.Metric, res); err != nil {
		return fmt.Errorf("failed to initialize metric provider: %w", err)
	}

	return nil
}

// createResource 创建 OpenTelemetry 资源
func (p *Provider) createResource(serviceName, serviceVersion string) (*resource.Resource, error) {

	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
		resource.WithService(),
		resource.WithFromEnv(),
		resource.WithHost(),
		// resource.WithProcess(),
		// resource.WithTelemetrySDK(),

	)
}

// initPropagator 初始化传播器
func (p *Provider) initPropagator() {
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
}

// initLog 初始化日志提供者
func (p *Provider) initLog(logConfig *LogConfig, res *resource.Resource) error {
	if !logConfig.Enable {
		return nil
	}

	logExporter, err := p.createLogExporter(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create log exporter: %w", err)
	}

	p.logProvider = log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)

	global.SetLoggerProvider(p.logProvider)

	// 设置全局 slog
	logger := otelslog.NewLogger(
		"global",
		otelslog.WithSource(true),
	)
	slog.SetDefault(logger)

	slog.Info("log provider initialized", "exporter", logConfig.Type)
	return nil
}

// createLogExporter 创建日志导出器
func (p *Provider) createLogExporter(logConfig *LogConfig) (log.Exporter, error) {
	switch logConfig.Type {
	case ExporterTypeStdout:
		if logConfig.Pretty {
			return stdoutlog.New(
				stdoutlog.WithPrettyPrint(),
			)
		}
		return stdoutlog.New()

	case ExporterTypeHTTP:
		return otlploghttp.New(
			context.Background(),
			otlploghttp.WithEndpointURL(logConfig.RemoteAddr),
			otlploghttp.WithHeaders(logConfig.Headers),
			otlploghttp.WithInsecure(),
		)

	case ExporterTypeGRPC:
		return otlploggrpc.New(
			context.Background(),
			otlploggrpc.WithEndpoint(logConfig.RemoteAddr),
			otlploggrpc.WithInsecure(),
			otlploggrpc.WithHeaders(logConfig.Headers),
		)

	default:
		return nil, fmt.Errorf("unsupported log exporter type: %s", logConfig.Type)
	}
}

// initTrace 初始化追踪提供者
func (p *Provider) initTrace(traceConfig TraceConfig, res *resource.Resource) error {
	if !traceConfig.Enable {
		return nil
	}

	traceExporter, err := createTraceExporter(traceConfig)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := trace.ParentBased(trace.TraceIDRatioBased(traceConfig.SamplingRatio))
	p.traceProvider = trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(p.traceProvider)

	slog.Info("trace provider initialized", "exporter", traceConfig.Type)
	return nil
}

// createTraceExporter 创建追踪导出器
func createTraceExporter(traceConfig TraceConfig) (trace.SpanExporter, error) {
	switch traceConfig.Type {
	case ExporterTypeStdout:
		if traceConfig.Pretty {
			return stdouttrace.New(
				stdouttrace.WithPrettyPrint(),
			)
		}
		return stdouttrace.New()

	case ExporterTypeHTTP:
		return otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpointURL(traceConfig.RemoteAddr),
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithHeaders(traceConfig.Headers),
		)

	case ExporterTypeGRPC:
		return otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(traceConfig.RemoteAddr),
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithHeaders(traceConfig.Headers),
		)

	default:
		return nil, fmt.Errorf("unsupported trace exporter type: %s", traceConfig.Type)
	}
}

// initMetric 初始化指标提供者
func (p *Provider) initMetric(metricConfig MetricConfig, res *resource.Resource) error {
	if !metricConfig.Enable {
		return nil
	}

	var meterProvider *metric.MeterProvider

	if metricConfig.Type == ExporterTypePrometheus {
		promeExporter, err := prometheus.New()
		if err != nil {
			return fmt.Errorf("failed to create prometheus exporter: %w", err)
		}

		meterProvider = metric.NewMeterProvider(
			metric.WithReader(promeExporter),
			metric.WithResource(res),
			metric.WithExemplarFilter(exemplar.TraceBasedFilter),
		)

	} else {
		metricExporter, err := createMetricExporter(metricConfig)
		if err != nil {
			return fmt.Errorf("failed to create metric exporter: %w", err)
		}

		meterProvider = metric.NewMeterProvider(
			metric.WithReader(
				metric.NewPeriodicReader(
					metricExporter,
					metric.WithInterval(time.Duration(metricConfig.IntervalSeconds)*time.Second),
				),
			),
			metric.WithResource(res),
			metric.WithExemplarFilter(exemplar.TraceBasedFilter),
		)
	}

	p.metricProvider = meterProvider
	otel.SetMeterProvider(meterProvider)

	// 启用 Go Runtime 指标
	if metricConfig.EnableRuntimeMetrics {
		if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(10 * time.Second)); err != nil {
			return fmt.Errorf("failed to start runtime metrics: %w", err)
		}
	}

	slog.Info("metric provider initialized", "exporter", metricConfig.Type)
	return nil
}

// createMetricExporter 创建指标导出器
func createMetricExporter(metricConfig MetricConfig) (metric.Exporter, error) {
	switch metricConfig.Type {
	case ExporterTypeStdout:
		if metricConfig.Pretty {
			return stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		}
		return stdoutmetric.New()

	case ExporterTypeHTTP:
		return otlpmetrichttp.New(
			context.Background(),
			otlpmetrichttp.WithEndpointURL(metricConfig.RemoteAddr),
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithHeaders(metricConfig.Headers),
		)

	case ExporterTypeGRPC:
		return otlpmetricgrpc.New(
			context.Background(),
			otlpmetricgrpc.WithEndpoint(metricConfig.RemoteAddr),
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithHeaders(metricConfig.Headers),
		)

	default:
		return nil, fmt.Errorf("unsupported metric exporter type: %s", metricConfig.Type)
	}
}

// Shutdown 关闭所有提供者
func (p *Provider) Shutdown(ctx context.Context) error {
	var errors []error

	if p.logProvider != nil {
		_ = p.logProvider.ForceFlush(ctx)
		if err := p.logProvider.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("log provider shutdown: %w", err))
		}
	}

	if p.traceProvider != nil {
		_ = p.traceProvider.ForceFlush(ctx)
		if err := p.traceProvider.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("trace provider shutdown: %w", err))
		}
	}

	if p.metricProvider != nil {
		_ = p.metricProvider.ForceFlush(ctx)
		if err := p.metricProvider.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("metric provider shutdown: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// GetLoggerProvider 获取日志提供者
func (p *Provider) GetLoggerProvider() *log.LoggerProvider {
	return p.logProvider
}

// GetTracerProvider 获取追踪提供者
func (p *Provider) GetTracerProvider() *trace.TracerProvider {
	return p.traceProvider
}

// GetMeterProvider 获取指标提供者
func (p *Provider) GetMeterProvider() *metric.MeterProvider {
	return p.metricProvider
}
