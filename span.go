package datadog

//go:generate go run github.com/tinylib/msgp -marshal=false -o=msgp.go -tests=false

import (
	"encoding/binary"

	traceAPI "go.opentelemetry.io/otel/api/trace"
	traceSDK "go.opentelemetry.io/otel/sdk/export/trace"
)

// Span is a DataDog Span
type Span struct {
	SpanID   uint64             `msg:"span_id"`
	TraceID  uint64             `msg:"trace_id"`
	ParentID uint64             `msg:"parent_id"`
	Name     string             `msg:"name"`
	Service  string             `msg:"service"`
	Resource string             `msg:"resource"`
	Type     string             `msg:"type"`
	Start    int64              `msg:"start"`
	Duration int64              `msg:"duration"`
	Meta     map[string]string  `msg:"meta,omitempty"`
	Metrics  map[string]float64 `msg:"metrics,omitempty"`
	Error    int32              `msg:"error"`
}

// ConvertSpan converts an OpenTelemetry span to a DataDog span.
func ConvertSpan(s *traceSDK.SpanData) *Span {
	span := &Span{
		TraceID:  binary.BigEndian.Uint64(s.SpanContext.TraceID[8:]),
		SpanID:   binary.BigEndian.Uint64(s.SpanContext.SpanID[:]),
		Name:     "opentelemetry",
		Resource: s.Name,
		Start:    s.StartTime.UnixNano(),
		Duration: s.EndTime.Sub(s.StartTime).Nanoseconds(),
		Metrics:  map[string]float64{},
		Meta:     map[string]string{},
	}

	if s.ParentSpanID.IsValid() {
		span.ParentID = binary.BigEndian.Uint64(s.ParentSpanID[:])
	}

	switch s.SpanKind {
	case traceAPI.SpanKindClient:
		span.Type = "client"
	case traceAPI.SpanKindServer:
		span.Type = "server"
	case traceAPI.SpanKindProducer:
		span.Type = "producer"
	case traceAPI.SpanKindConsumer:
		span.Type = "consumer"
	}

	return span
}
