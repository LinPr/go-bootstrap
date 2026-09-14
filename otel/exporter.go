package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

// createLogExporter creates a log exporter.
func (p *OtelProviders) createLogExporter(logConfig *LogConfig) (log.Exporter, error) {
	switch logConfig.Exporter {
	case ExporterTypeFile:
		opts := []stdoutlog.Option{
			stdoutlog.WithWriter(
				&lumberjack.Logger{
					Filename:   logConfig.Rotate.Filename,
					MaxSize:    logConfig.Rotate.MaxMB,
					MaxAge:     logConfig.Rotate.MaxDay,
					MaxBackups: logConfig.Rotate.MaxBackups,
					LocalTime:  logConfig.Rotate.LocalTime,
					Compress:   logConfig.Rotate.Compress,
				}),
		}
		if logConfig.Pretty {
			opts = append(opts, stdoutlog.WithPrettyPrint())
		}
		return stdoutlog.New(opts...)

	case ExporterTypeStdout:
		if logConfig.Pretty {
			return stdoutlog.New(stdoutlog.WithPrettyPrint())
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
