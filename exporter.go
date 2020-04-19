package datadog

import (
	"time"

	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/ungrouped"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Exporter exports OpenTelemetry traces and metrics to DataDog.
type Exporter struct {

	// Meter
	pusher *push.Controller
}

// NewExporter returns a new DataDog OpenTelemetry Exporter
func NewExporter() (*Exporter, error) {
	exp, err := NewMeterExpoter()
	if err != nil {
		return nil, err
	}

	selector := simple.NewWithExactMeasure()

	batcher := ungrouped.New(selector, export.NewDefaultLabelEncoder(), true)

	pusher := push.New(batcher, exp, time.Minute)
	pusher.Start()

	return &Exporter{
		pusher: pusher,
	}, nil
}
