package datadog

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/sdk/export/trace"
)

// ExportSpan receives a single span and exports it to DataDog
// TODO: implementation detail
func (e *Exporter) ExportSpan(ctx context.Context, span *trace.SpanData) {
	log.Printf("Export Span: %+v\n", span)
}

// ExportSpans receives a multiple spans and exports them to DataDog
// TODO: implementation detail
func (e *Exporter) ExportSpans(ctx context.Context, spans []*trace.SpanData) {
	log.Printf("Export Spans\n")
	for _, span := range spans {
		e.ExportSpan(ctx, span)
	}
}
