package datadog

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/api/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func InstallNewPipeline() (*push.Controller, error) {
	controller, err := NewExportPipeline()
	if err != nil {
		return controller, err
	}

	global.SetMeterProvider(controller)

	return controller, err
}

func NewExportPipeline() (*push.Controller, error) {
	exp, err := NewMeterExpoter()
	if err != nil {
		return nil, err
	}

	selector := simple.NewWithExactMeasure()

	integrator := integrator.New(selector, export.NewDefaultLabelEncoder(), true)

	pusher := push.New(integrator, exp, time.Minute)
	pusher.Start()

	return pusher, nil
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
