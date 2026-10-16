package filelog

import (
	"io"
	"os"
)

type LogFormat string

const (
	LogFormatOtlpJson LogFormat = "otlp_json"
	LogFormatJson     LogFormat = "json"
)

var (
	defaultWriter io.Writer = os.Stdout
	defaultFormat LogFormat = LogFormatOtlpJson
)

// config contains options for the STDOUT exporter.
type config struct {
	// Writer is the destination. If not set, os.Stdout is used.
	Writer io.Writer
	Format LogFormat
}

// newConfig creates a config from options.
func newConfig(options []Option) config {
	cfg := config{
		Writer: defaultWriter,
		Format: defaultFormat,
	}
	for _, opt := range options {
		cfg = opt.apply(cfg)
	}
	return cfg
}

// Option sets the configuration value for an Exporter.
type Option interface {
	apply(config) config
}

// WithWriter sets the export stream destination.
func WithWriter(w io.Writer) Option {
	return writerOption{w}
}

type writerOption struct {
	Writer io.Writer
}

// WithFormat sets the log format.
func WithFormat(format LogFormat) Option {
	return formatOption{format}
}

type formatOption struct {
	Format LogFormat
}

func (o formatOption) apply(cfg config) config {
	cfg.Format = o.Format
	return cfg
}

func (o writerOption) apply(cfg config) config {
	cfg.Writer = o.Writer
	return cfg
}
