package otel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogr"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/bridges/otelzap"
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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// OtelProviders holds the OpenTelemetry providers.
type OtelProviders struct {
	logProvider    *log.LoggerProvider
	traceProvider  *trace.TracerProvider
	metricProvider *metric.MeterProvider
}

// newOtelProviders creates a new OpenTelemetry provider set.
func newOtelProviders(config *Config) (*OtelProviders, error) {
	if config == nil {
		config = DefaultConfig()
	}

	p := &OtelProviders{
		logProvider:    nil,
		traceProvider:  nil,
		metricProvider: nil,
	}

	if err := p.initialize(config); err != nil {
		return nil, err
	}

	return p, nil
}

// initialize initializes the OpenTelemetry SDK.
func (p *OtelProviders) initialize(c *Config) error {
	// Set up context propagation.
	p.initPropagator()

	// Create the resource.
	res, err := p.createResource(c.ServiceName, c.ServiceVersion)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize logs.
	if err := p.initLog(&c.Log, res); err != nil {
		return fmt.Errorf("failed to initialize log provider: %w", err)
	}

	// Initialize traces.
	if err := p.initTrace(c.Trace, res); err != nil {
		return fmt.Errorf("failed to initialize trace provider: %w", err)
	}

	// Initialize metrics.
	if err := p.initMetric(c.Metric, res); err != nil {
		return fmt.Errorf("failed to initialize metric provider: %w", err)
	}

	return nil
}

// createResource creates an OpenTelemetry resource.
func (p *OtelProviders) createResource(serviceName, serviceVersion string) (*resource.Resource, error) {

	return resource.New(
		context.Background(),
		resource.WithFromEnv(), // Read environment variables first.
		resource.WithHost(),    // Add host information.
		resource.WithAttributes( // Set service information last so it wins.
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
		// resource.WithProcess(),
		// resource.WithTelemetrySDK(),

	)
}

// initPropagator initializes the propagator.
func (p *OtelProviders) initPropagator() {
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
}

// initLog initializes the log provider.
func (p *OtelProviders) initLog(logConfig *LogConfig, res *resource.Resource) error {
	if !logConfig.Enable {
		return nil
	}

	logExporter, err := p.createLogExporter(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create log exporter: %w", err)
	}

	var processor log.Processor
	processor = log.NewBatchProcessor(
		logExporter,
	)
	if logConfig.Logger == LoggerTypeSlog {
		slogSeverity, _, _, err := parseLogLevel(logConfig.Level)
		if err != nil {
			return err
		}
		processor = newSeverityProcessor(
			processor,
			slogSeverity,
		)
	}

	p.logProvider = log.NewLoggerProvider(
		log.WithProcessor(
			processor,
		),
		log.WithResource(res),
		log.WithAttributeCountLimit(50),
	)

	global.SetLoggerProvider(p.logProvider)

	// Configure the global log bridge based on the selected logger type.
	if err := p.setupLoggerBridge(logConfig); err != nil {
		return fmt.Errorf("failed to setup logger bridge: %w", err)
	}

	slog.Info("log provider initialized", "exporter", string(logConfig.Exporter), "logger", string(logConfig.Logger))
	return nil
}

// setupLoggerBridge configures the log bridge.
func (p *OtelProviders) setupLoggerBridge(logConfig *LogConfig) error {

	_, zapLevel, logrusLevel, err := parseLogLevel(logConfig.Level)
	if err != nil {
		return err
	}

	switch logConfig.Logger {
	case LoggerTypeSlog:
		var handler slog.Handler
		handler = otelslog.NewHandler(
			"global",
			otelslog.WithLoggerProvider(p.logProvider),
			otelslog.WithSource(true),
		)

		// Wrap the otelslog handler so struct/map/slice attributes are JSON
		// encoded here, before the bridge flattens them via fmt %+v.
		if logConfig.Pretty {
			handler = newjsonHandler(handler, logConfig.Pretty)
		}

		logger := slog.New(handler)
		slog.SetDefault(logger)

	case LoggerTypeZap:
		zap.NewExample()
		logger := zap.New(
			otelzap.NewCore(
				"global",
				otelzap.WithLoggerProvider(p.logProvider),
			),
			zap.IncreaseLevel(zapLevel),
			zap.AddCaller(),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		zap.ReplaceGlobals(logger)

	case LoggerTypeLogrus:

		hook := otellogrus.NewHook(
			"global",
			otellogrus.WithLoggerProvider(p.logProvider),
		)
		logger := logrus.New()
		logger.AddHook(hook)
		logger.SetLevel(logrusLevel)

		// Set as the global logger.
		logrus.SetFormatter(logger.Formatter)
		logrus.SetOutput(logger.Out)
		logrus.SetLevel(logger.Level)
		logrus.AddHook(hook)

	case LoggerTypeLogr:
		logSink := otellogr.NewLogSink(
			"global",
			otellogr.WithLoggerProvider(p.logProvider),
		)
		// logr instances are managed by the caller; this only creates an example.
		loggger := logr.New(logSink)
		// TODO:
		_ = loggger     // Avoid unused warnings; applications can use this logger.
		_ = logrusLevel // otellogr bridge currently has no min-level option equivalent to slog/zap/logrus.
		// otel.SetLogger(loggger)
	default:
		return fmt.Errorf("unsupported logger type: %s", logConfig.Logger)
	}

	return nil
}

// createLogExporter creates a log exporter.
func (p *OtelProviders) createLogExporter(logConfig *LogConfig) (log.Exporter, error) {
	switch logConfig.Exporter {
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
		return nil, fmt.Errorf("unsupported log exporter type: %s", logConfig.Exporter)
	}
}

// initTrace initializes the trace provider.
func (p *OtelProviders) initTrace(traceConfig TraceConfig, res *resource.Resource) error {
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

	slog.Info("trace provider initialized", "exporter", string(traceConfig.Exporter))
	return nil
}

// createTraceExporter creates a trace exporter.
func createTraceExporter(traceConfig TraceConfig) (trace.SpanExporter, error) {
	switch traceConfig.Exporter {
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
		return nil, fmt.Errorf("unsupported trace exporter type: %s", traceConfig.Exporter)
	}
}

// initMetric initializes the metric provider.
func (p *OtelProviders) initMetric(metricConfig MetricConfig, res *resource.Resource) error {
	if !metricConfig.Enable {
		return nil
	}

	var meterProvider *metric.MeterProvider

	if metricConfig.Exporter == ExporterTypePrometheus {
		promeExporter, err := prometheus.New()
		if err != nil {
			return fmt.Errorf("failed to create prometheus exporter: %w", err)
		}

		meterProvider = metric.NewMeterProvider(
			metric.WithReader(promeExporter),
			metric.WithResource(res),
			metric.WithExemplarFilter(exemplar.TraceBasedFilter),
			// metric.WithCardinalityLimit(2000),
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

	// Enable Go runtime metrics.
	if metricConfig.EnableRuntimeMetrics {
		if err := runtime.Start(
			runtime.WithMinimumReadMemStatsInterval(time.Duration(metricConfig.IntervalSeconds) * time.Second),
		); err != nil {
			return fmt.Errorf("failed to start runtime metrics: %w", err)
		}
	}

	slog.Info("metric provider initialized", "exporter", string(metricConfig.Exporter))
	return nil
}

// createMetricExporter creates a metric exporter.
func createMetricExporter(metricConfig MetricConfig) (metric.Exporter, error) {
	switch metricConfig.Exporter {
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
		return nil, fmt.Errorf("unsupported metric exporter type: %s", metricConfig.Exporter)
	}
}

// Shutdown closes all providers.
func (p *OtelProviders) Shutdown(ctx context.Context) error {
	var errs []error

	if p.logProvider != nil {
		_ = p.logProvider.ForceFlush(ctx)
		if err := p.logProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("log provider shutdown: %w", err))
		}
	}

	if p.traceProvider != nil {
		_ = p.traceProvider.ForceFlush(ctx)
		if err := p.traceProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("trace provider shutdown: %w", err))
		}
	}

	if p.metricProvider != nil {
		_ = p.metricProvider.ForceFlush(ctx)
		if err := p.metricProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("metric provider shutdown: %w", err))
		}
	}

	return errors.Join(errs...)
}
