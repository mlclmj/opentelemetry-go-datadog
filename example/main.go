package main

import (
	"context"
	"log"
	"time"

	datadog "go.krak3n.codes/opentelemetry-go-datadog"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	sleep = time.Millisecond * 100
)

func main() {
	// Initialise context
	ctx := context.Background()

	// Initialise Exporter
	exp, err := datadog.NewExporter()
	if err != nil {
		log.Fatal(err)
	}

	defer exp.Stop()

	// Set tracing Provider
	// TODO: move to constructor
	provider, err := sdktrace.NewProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}))
	if err != nil {
		log.Fatal(err)
	}

	// Register trace provider
	// TODO: move to constructor
	global.SetTraceProvider(provider)

	// Register metric provider
	// TODO: move to constructor
	global.SetMeterProvider(exp)

	tracer := global.Tracer("ex.com/basic")
	meter := global.Meter("ex.com/basic")

	counter1 := metric.Must(meter).NewInt64Counter("ex.com/counter1")
	measure1 := metric.Must(meter).NewInt64Measure("ex.com/measure1")

	err = tracer.WithSpan(ctx, "operation", func(ctx context.Context) error {
		log.Println("operation")
		time.Sleep(sleep)

		counter1.Add(ctx, 1)
		measure1.Record(ctx, 1)

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
