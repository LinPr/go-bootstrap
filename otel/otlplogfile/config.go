package otlplogfile

import (
	"io"
	"os"
)

var (
	defaultWriter io.Writer = os.Stdout
)

// config contains options for the STDOUT exporter.
type config struct {
	// Writer is the destination. If not set, os.Stdout is used.
	Writer io.Writer
}

// newConfig creates a config from options.
func newConfig(options []Option) config {
	cfg := config{
		Writer: defaultWriter,
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
	W io.Writer
}

func (o writerOption) apply(cfg config) config {
	cfg.Writer = o.W
	return cfg
}
