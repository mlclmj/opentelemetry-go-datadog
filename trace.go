package datadog

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/sdk/export/trace"
)

// TraceExporter exports traces to DataDog.
type TraceExporter struct {
}

// NewTraceExporter constructs a new TraceExporter.
func NewTraceExporter() (*TraceExporter, error) {
	return &TraceExporter{}, nil
}

// ExportSpan receives a single span and exports it to DataDog
// TODO: implementation detail
func (e *TraceExporter) ExportSpan(ctx context.Context, span *trace.SpanData) {
	log.Printf("Export Span: %+v\n", span)
}

// ExportSpans receives a multiple spans and exports them to DataDog
// TODO: implementation detail
func (e *TraceExporter) ExportSpans(ctx context.Context, spans []*trace.SpanData) {
	log.Printf("Export Spans\n")
	for _, span := range spans {
		e.ExportSpan(ctx, span)
	}
}
