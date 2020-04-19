package datadog

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
)

// Meter returns a meter with the given name
func (e *Exporter) Meter(name string) metric.Meter {
	return e.pusher.Meter(name)
}

// Stop stops the exporter
func (e *Exporter) Stop() {
	if e.pusher == nil {
		return
	}

	e.pusher.Stop()
}

// MeterExporter exports metrics to DataDog
type MeterExporter struct {
}

// NewMeterExpoter constructs a NewMeterExpoter
func NewMeterExpoter() (*MeterExporter, error) {
	return &MeterExporter{}, nil
}

// Export exports the provide metric record to DataDog.
func (e *MeterExporter) Export(ctx context.Context, checkpoint export.CheckpointSet) error {
	log.Printf("Export Checkpoint: %+v\n", checkpoint)
	return nil
}
