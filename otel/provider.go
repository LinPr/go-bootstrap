package otel

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
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
	conf := DefaultConfig()
	if config != nil {
		// Top-level fields.
		conf.ServiceName = cmp.Or(config.ServiceName, conf.ServiceName)
		conf.ServiceVersion = cmp.Or(config.ServiceVersion, conf.ServiceVersion)

		// Log config.
		conf.Log.Enable = config.Log.Enable
		conf.Log.Exporter = cmp.Or(config.Log.Exporter, conf.Log.Exporter)
		conf.Log.Logger = cmp.Or(config.Log.Logger, conf.Log.Logger)
		conf.Log.Level = cmp.Or(config.Log.Level, conf.Log.Level)
		conf.Log.RemoteAddr = cmp.Or(config.Log.RemoteAddr, conf.Log.RemoteAddr)
		if config.Log.Headers != nil {
			conf.Log.Headers = config.Log.Headers
		}
		conf.Log.Pretty = config.Log.Pretty
		conf.Log.AttributeCountLimit = cmp.Or(config.Log.AttributeCountLimit, conf.Log.AttributeCountLimit)
		conf.Log.Rotate.Filename = cmp.Or(config.Log.Rotate.Filename, conf.Log.Rotate.Filename)
		conf.Log.Rotate.MaxMB = cmp.Or(config.Log.Rotate.MaxMB, conf.Log.Rotate.MaxMB)
		conf.Log.Rotate.MaxDay = cmp.Or(config.Log.Rotate.MaxDay, conf.Log.Rotate.MaxDay)
		conf.Log.Rotate.MaxBackups = cmp.Or(config.Log.Rotate.MaxBackups, conf.Log.Rotate.MaxBackups)
		conf.Log.Rotate.LocalTime = config.Log.Rotate.LocalTime
		conf.Log.Rotate.Compress = config.Log.Rotate.Compress

		// Trace config.
		conf.Trace.Enable = config.Trace.Enable
		conf.Trace.Exporter = cmp.Or(config.Trace.Exporter, conf.Trace.Exporter)
		conf.Trace.RemoteAddr = cmp.Or(config.Trace.RemoteAddr, conf.Trace.RemoteAddr)
		if config.Trace.Headers != nil {
			conf.Trace.Headers = config.Trace.Headers
		}
		conf.Trace.Pretty = config.Trace.Pretty
		conf.Trace.SamplingRatio = cmp.Or(config.Trace.SamplingRatio, conf.Trace.SamplingRatio)

		// Metric config.
		conf.Metric.Enable = config.Metric.Enable
		conf.Metric.Exporter = cmp.Or(config.Metric.Exporter, conf.Metric.Exporter)
		conf.Metric.RemoteAddr = cmp.Or(config.Metric.RemoteAddr, conf.Metric.RemoteAddr)
		if config.Metric.Headers != nil {
			conf.Metric.Headers = config.Metric.Headers
		}
		conf.Metric.Pretty = config.Metric.Pretty
		conf.Metric.IntervalSeconds = cmp.Or(config.Metric.IntervalSeconds, conf.Metric.IntervalSeconds)
		conf.Metric.EnableRuntimeMetrics = config.Metric.EnableRuntimeMetrics
		conf.Metric.CardinalityLimit = cmp.Or(config.Metric.CardinalityLimit, conf.Metric.CardinalityLimit)
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
		resource.WithAttributes( // Set service information last so it wins.
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
		// resource.WithHost(),
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
		slogSeverity, _, err := parseLogLevel(logConfig.Level)
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
		log.WithAttributeCountLimit(logConfig.AttributeCountLimit),
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

	_, zapLevel, err := parseLogLevel(logConfig.Level)
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

	default:
		return fmt.Errorf("unsupported logger type: %s", logConfig.Logger)
	}

	return nil
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
			metric.WithCardinalityLimit(metricConfig.CardinalityLimit),
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
