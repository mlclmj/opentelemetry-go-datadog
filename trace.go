package datadog

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tinylib/msgp/msgp"
	"go.opentelemetry.io/otel/sdk/export/trace"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	"google.golang.org/api/support/bundler"
)

// DataDog agent constants.
const (
	PacketLimit    = int(1e7) // 10MB
	FlushThreshold = PacketLimit / 2
)

// Default HTTP Client Timeouts and Settings.
const (
	OneSecondTimeout    = time.Second * 1
	TenSecondTimeout    = time.Second * 10
	ThirtySecondTimeout = time.Second * 30
	NintySecondTimeout  = time.Second * 90
	DefaultMaxIdleConns = 100
)

// MessagePack constants.
const (
	MsgPackMaxLength      = uint(math.MaxUint32)
	MsgPackArrayFix  byte = 144  // up to 15 items
	MsgPackArray16        = 0xdc // up to 2^16-1 items, followed by size in 2 bytes
	MsgPackArray32        = 0xdd // up to 2^32-1 items, followed by size in 4 bytes
)

// Ensure TraceExporter satisfies the export.SpanSyncer interface.
var _ export.SpanSyncer = (*TraceExporter)(nil)

// An Uploader uploads trace data to the DataDog trace agent.
type Uploader interface {
	Upload(data io.Reader, count int) (TraceAgentResponse, error)
}

// TraceExporter exports traces to DataDog.
type TraceExporter struct {
	bundler *bundler.Bundler
}

// NewTraceExporter constructs a new TraceExporter.
func NewTraceExporter() (*TraceExporter, error) {
	uploader := NewTraceAgent()

	bundler := bundler.NewBundler((*Span)(nil), func(bundle interface{}) {
		spans := bundle.([]*Span)

		req := NewTraceAgentRequest()

		for _, span := range spans {
			if err := req.Add(span); err != nil {
				log.Println(err)
			}

			if req.Size() > FlushThreshold {
				_, err := uploader.Upload(req.Buffer(), 1)
				if err != nil {
					log.Println(err)
					return
				}

				req.Reset()
			}
		}

		_, err := uploader.Upload(req.Buffer(), 1)
		if err != nil {
			log.Println(err)
			return
		}

		req.Reset()
	})

	return &TraceExporter{
		bundler: bundler,
	}, nil
}

// ExportSpan receives a single span and exports it to DataDog
func (e *TraceExporter) ExportSpan(ctx context.Context, span *trace.SpanData) {
	if err := e.bundler.Add(ConvertSpan(span), 1); err != nil {
		log.Println(err)
	}
}

// ExportSpans receives a multiple spans and exports them to DataDog
func (e *TraceExporter) ExportSpans(ctx context.Context, spans []*trace.SpanData) {
	for _, span := range spans {
		e.ExportSpan(ctx, span)
	}
}

// Flush exports spans to DataDog.
func (e *TraceExporter) Flush() {
	e.bundler.Flush()
}

// SpanPackets holds the spans encoded in msgpack format. Spans are added squentially whilst
// keeping track of the count.
type SpanPackets struct {
	count uint64
	data  bytes.Buffer
}

// Add adds a span to SpanPackets.
func (s *SpanPackets) Add(span *Span) error {
	if uint(s.count) >= MsgPackMaxLength {
		return ErrMsgPackOverflow
	}

	if err := msgp.Encode(&s.data, span); err != nil {
		return err
	}

	s.count++

	return nil
}

// Size returns the size of the packets including the msgpack headers.
func (s *SpanPackets) Size() int {
	return s.data.Len() + msgpHeaderSize(s.count)
}

// Bytes returns msgpack encoded spans without the headers.
func (s *SpanPackets) Bytes() []byte {
	var header [8]byte

	off := msgpHeader(&header, s.count)

	var buf bytes.Buffer

	buf.Write(header[off:])
	buf.Write(s.data.Bytes())

	return buf.Bytes()
}

// Reset clears spans and sets the count to 0.
func (s *SpanPackets) Reset() {
	s.count = 0
	s.data.Reset()
}

// TraceAgentRequest holds traces waiting to be sent the DataDog agent.
type TraceAgentRequest struct {
	packets map[uint64]*SpanPackets
	size    int
}

// NewTraceAgentRequest constructs anew TraceAgentRequest
func NewTraceAgentRequest() *TraceAgentRequest {
	return &TraceAgentRequest{
		packets: make(map[uint64]*SpanPackets),
	}
}

// Add adds a span to the TraceAgentRequest packets.
func (r TraceAgentRequest) Add(span *Span) error {
	if uint(len(r.packets)) >= MsgPackMaxLength {
		return ErrMsgPackOverflow
	}

	tid := span.TraceID
	if _, ok := r.packets[tid]; !ok {
		r.packets[tid] = new(SpanPackets)
	}

	old := r.packets[tid].Size()
	if err := r.packets[tid].Add(span); err != nil {
		return err
	}

	size := r.packets[tid].Size()

	r.size += size - old

	return nil
}

// Size returns the size of the request including the headers.
func (r *TraceAgentRequest) Size() int {
	return r.size + msgpHeaderSize(uint64(len(r.packets)))
}

// Buffer returns a copy of the msgpack encoded packets.
func (r *TraceAgentRequest) Buffer() *bytes.Buffer {
	var header [8]byte
	var buffer bytes.Buffer

	off := msgpHeader(&header, uint64(len(r.packets)))

	buffer.Write(header[off:])

	for _, packet := range r.packets {
		buffer.Write(packet.Bytes())
	}

	return &buffer
}

// Reset resets the trace request.
func (r *TraceAgentRequest) Reset() {
	r.packets = make(map[uint64]*SpanPackets)
	r.size = 0
}

// TraceAgentResponse is the response from the DataDog trace agent after successful upload.
type TraceAgentResponse struct {
	Rates map[string]float64 `json:"rate_by_service"`
}

// TraceAgent uploads traces to the DataDog trace agent.
type TraceAgent struct {
	url     *url.URL
	client  *http.Client
	headers map[string]string
}

// NewTraceAgent constructs a new TraceAgent.
func NewTraceAgent() *TraceAgent {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   ThirtySecondTimeout,
				KeepAlive: ThirtySecondTimeout,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          DefaultMaxIdleConns,
			IdleConnTimeout:       NintySecondTimeout,
			TLSHandshakeTimeout:   TenSecondTimeout,
			ExpectContinueTimeout: OneSecondTimeout,
		},
		Timeout: OneSecondTimeout,
	}

	url := &url.URL{
		Scheme: "http",
		Host:   "localhost:8126",
		Path:   path.Join("v0.4", "traces"),
	}

	headers := map[string]string{
		"Datadog-Meta-Lang":               "go",
		"Datadog-Meta-Lang-Version":       strings.TrimPrefix(runtime.Version(), "go"),
		"Datadog-Meta-Lang-Interpreter":   runtime.Compiler + "-" + runtime.GOARCH + "-" + runtime.GOOS,
		"Datadog-Meta-TraceAgent-Version": "OTEL/v0.4.2",
		"Content-Type":                    "application/msgpack",
	}

	return &TraceAgent{
		url:     url,
		client:  client,
		headers: headers,
	}
}

// Upload uploads a trace to the DataDog trace agent.
func (t *TraceAgent) Upload(data io.Reader, count int) (TraceAgentResponse, error) {
	var tar TraceAgentResponse

	req, err := http.NewRequest(http.MethodPost, t.url.String(), data)
	if err != nil {
		return tar, err
	}

	req.Header.Set("X-DataDog-Trace-Count", strconv.Itoa(count))
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	rsp, err := t.client.Do(req)
	if err != nil {
		return tar, err
	}

	defer Close(rsp.Body)

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return tar, err
	}

	if rsp.StatusCode >= http.StatusBadRequest {
		return tar, fmt.Errorf("[%d %s] %s",
			rsp.StatusCode,
			http.StatusText(rsp.StatusCode),
			string(body))
	}

	if err := json.Unmarshal(body, &tar); err != nil {
		return tar, err
	}

	return tar, nil
}

// msgpHeaderSize returns the size in bytes of a header for a msgpack array of length n.
func msgpHeaderSize(n uint64) int {
	switch {
	case n == 0:
		return 0
	case n <= 15:
		return 1
	case n <= math.MaxUint16:
		return 3
	case n <= math.MaxUint32:
		fallthrough
	default:
		return 5
	}
}

// msgpHeader writes the msgpack array header for a slice of length n into out.
// It returns the offset at which to begin reading from out. For more information,
// see the msgpack spec:
// https://github.com/msgpack/msgpack/blob/master/spec.md#array-format-family
func msgpHeader(out *[8]byte, n uint64) (off int) {
	off = 8 - msgpHeaderSize(n)
	switch {
	case n <= 15:
		out[off] = MsgPackArrayFix + byte(n)
	case n <= math.MaxUint16:
		binary.BigEndian.PutUint64(out[:], n) // writes 2 bytes
		out[off] = MsgPackArray16
	case n <= math.MaxUint32:
		fallthrough
	default:
		binary.BigEndian.PutUint64(out[:], n) // writes 4 bytes
		out[off] = MsgPackArray32
	}
	return off
}
