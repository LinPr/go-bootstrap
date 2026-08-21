package otel

import (
	"context"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

// severityProcessor wraps a downstream log.Processor and drops records whose
// severity is below minSeverity. By also implementing Enabled it lets the
// bridge skip building records below the floor. Modeled on the upstream
// minsev.LogProcessor; the floor comes from the injected log level config.
type severityProcessor struct {
	log.Processor
	minSeverity otellog.Severity
}

var _ log.Processor = (*severityProcessor)(nil)

// newSeverityProcessor wraps downstream with a minimum severity floor.
func newSeverityProcessor(downstream log.Processor, minSeverity otellog.Severity) *severityProcessor {
	return &severityProcessor{
		Processor:   downstream,
		minSeverity: minSeverity,
	}
}

// OnEmit forwards the record downstream only when it meets the severity floor.
func (p *severityProcessor) OnEmit(ctx context.Context, record *log.Record) error {
	if record.Severity() >= p.minSeverity {
		return p.Processor.OnEmit(ctx, record)
	}
	return nil
}

// Enabled reports enabled only when the severity meets the floor and the
// wrapped processor also agrees.
func (p *severityProcessor) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	return param.Severity >= p.minSeverity && p.Processor.Enabled(ctx, param)
}
