package tracing

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// ---- spanRecord projection ----

// TestBuildSpanRecord_NoAttrs covers the early-return branch where the
// span has no attributes; the Attributes map must be left nil so the
// JSON tag `omitempty` strips it from the wire payload.
func TestBuildSpanRecord_NoAttrs(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "no-attrs-span")
	span.End()
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 recorded span, got %d", len(spans))
	}
	rec := buildSpanRecord(spans[0])
	if rec.Name != "no-attrs-span" {
		t.Fatalf("unexpected name: %q", rec.Name)
	}
	if rec.TraceID == "" {
		t.Fatal("expected non-empty trace id")
	}
	if rec.SpanID == "" {
		t.Fatal("expected non-empty span id")
	}
	if rec.StartTime == "" {
		t.Fatal("expected non-empty start time")
	}
	if rec.Attributes != nil {
		t.Fatalf("expected nil attributes map, got %v", rec.Attributes)
	}
}

// TestBuildSpanRecord_WithAttrs covers the attribute-flattening branch.
// kv.Value.Emit() must be used so non-string attribute values round-trip
// to their canonical string form.
func TestBuildSpanRecord_WithAttrs(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "with-attrs-span",
		trace.WithAttributes(
			attribute.String("k1", "v1"),
			attribute.Int("k2", 42),
		),
	)
	span.End()
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 recorded span, got %d", len(spans))
	}
	rec := buildSpanRecord(spans[0])
	if len(rec.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d (%v)", len(rec.Attributes), rec.Attributes)
	}
	if rec.Attributes["k1"] != "v1" {
		t.Fatalf("expected k1=v1, got %q", rec.Attributes["k1"])
	}
	if rec.Attributes["k2"] != "42" {
		t.Fatalf("expected k2=42 (Emit form), got %q", rec.Attributes["k2"])
	}
}

// TestBuildSpanRecord_JSONShape pins the on-the-wire JSON contract. Any
// silent rename of a field here is a breaking change for downstream
// consumers, so this test fails loudly.
func TestBuildSpanRecord_JSONShape(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "wire-shape",
		trace.WithAttributes(attribute.String("k", "v")),
	)
	span.End()

	rec := buildSpanRecord(sr.Ended()[0])
	payload, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"trace_id"`, `"span_id"`, `"name"`, `"start_time"`, `"attributes"`, `"k":"v"`} {
		if !contains(payload, key) {
			t.Fatalf("expected payload to contain %s, got: %s", key, payload)
		}
	}
}

func contains(haystack []byte, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack []byte, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == needle {
			return i
		}
	}
	return -1
}

// ---- redisSpanRecord projection ----

// TestBuildRedisSpanRecord_NoAttrs mirrors the kafka projection test.
func TestBuildRedisSpanRecord_NoAttrs(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "redis-no-attrs")
	span.End()
	rec := buildRedisSpanRecord(sr.Ended()[0])
	if rec.Attributes != nil {
		t.Fatalf("expected nil attributes, got %v", rec.Attributes)
	}
}

// TestBuildRedisSpanRecord_WithAttrs covers the attribute branch.
func TestBuildRedisSpanRecord_WithAttrs(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "redis-attrs",
		trace.WithAttributes(attribute.String("coll", "users")),
	)
	span.End()
	rec := buildRedisSpanRecord(sr.Ended()[0])
	if rec.Attributes["coll"] != "users" {
		t.Fatalf("expected coll=users, got %q", rec.Attributes["coll"])
	}
}

// ---- redisStreamClient stub for ExportSpans ----

// stubRedisStreamClient is a minimal redisStreamClient implementation
// for unit-testing ExportSpans without a real Redis. It records every
// XAdd invocation and lets the test inject an error to exercise the
// error-wrapping branch.
type stubRedisStreamClient struct {
	mu     sync.Mutex
	calls  []*redis.XAddArgs
	err    error
	closed bool
}

func (s *stubRedisStreamClient) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, args)
	cmd := redis.NewStringCmd(ctx)
	if s.err != nil {
		cmd.SetErr(s.err)
	} else {
		cmd.SetVal("0-0")
	}
	return cmd
}

func (s *stubRedisStreamClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// TestRedisStreamExporter_ExportSpans_WithSpans covers the inner loop:
// build-record → marshal → XAdd, with the stub verifying call counts
// and the "span" field key.
func TestRedisStreamExporter_ExportSpans_WithSpans(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "user-action",
		trace.WithAttributes(attribute.String("u", "alice")),
	)
	span.End()

	stub := &stubRedisStreamClient{}
	exp := &redisStreamExporter{client: stub, stream: "otel"}

	if err := exp.ExportSpans(context.Background(), sr.Ended()); err != nil {
		t.Fatalf("ExportSpans: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 XAdd call, got %d", len(stub.calls))
	}
	if stub.calls[0].Stream != "otel" {
		t.Fatalf("expected stream=otel, got %q", stub.calls[0].Stream)
	}
	// Values is interface{} (the redis driver accepts multiple shapes);
	// the production code passes map[string]interface{}{"span": payload}.
	valuesMap, ok := stub.calls[0].Values.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Values to be map[string]interface{}, got %T", stub.calls[0].Values)
	}
	payload, ok := valuesMap["span"].([]byte)
	if !ok {
		t.Fatalf("expected []byte payload under key \"span\", got %T", valuesMap["span"])
	}
	if !contains(payload, `"name":"user-action"`) {
		t.Fatalf("expected payload to contain span name, got: %s", payload)
	}
	if !contains(payload, `"u":"alice"`) {
		t.Fatalf("expected payload to contain flattened attribute, got: %s", payload)
	}
}

// TestRedisStreamExporter_ExportSpans_PropagatesError covers the
// error-wrapping branch when the underlying XAdd fails.
func TestRedisStreamExporter_ExportSpans_PropagatesError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "boom")
	span.End()

	stub := &stubRedisStreamClient{err: context.DeadlineExceeded}
	exp := &redisStreamExporter{client: stub, stream: "x"}

	err := exp.ExportSpans(context.Background(), sr.Ended())
	if err == nil {
		t.Fatal("expected XAdd failure to surface")
	}
	if err.Error() == "" || (err.Error() != "tracing: redis_stream: XAdd: context deadline exceeded" &&
		!contains([]byte(err.Error()), "redis_stream: XAdd")) {
		t.Fatalf("error should wrap redis_stream: XAdd, got: %v", err)
	}
}

// ---- kafka Sender stub for ExportSpans ----

// stubKafkaSender is a minimal kafkaSender implementation that records
// every Send call. It returns the pre-set error verbatim so the
// produce-error branch in ExportSpans can be exercised.
type stubKafkaSender struct {
	mu      sync.Mutex
	calls   []*kafka.Message
	err     error
	flushN  int // how many in-flight messages Flush should report
	flushOK bool
}

func (s *stubKafkaSender) Send(msg *kafka.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, msg)
	return s.err
}

// Flush mirrors librdkafka's behaviour: returns the number of still-
// in-flight messages. The stub always reports zero (drained) unless
// flushN is explicitly set via SetFlushPending.
func (s *stubKafkaSender) Flush(timeoutMs int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = timeoutMs
	if s.flushOK {
		return 0
	}
	return s.flushN
}

func (s *stubKafkaSender) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Close the underlying kafka producer channel mock surface — in
	// tests we don't ship an Events channel, so this is a no-op.
}

// SetFlushPending configures the stub to report `n` pending messages
// from the next Flush() call. Used to drive the partial-flush error
// branch in Shutdown.
func (s *stubKafkaSender) SetFlushPending(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushN = n
	s.flushOK = false
}

// TestKafkaSenderStub_PassesThrough is a sanity check on the stub
// itself; if this fails the higher-level ExportSpans tests would
// observe phantom behaviour.
func TestKafkaSenderStub_PassesThrough(t *testing.T) {
	stub := &stubKafkaSender{}
	topic := "t"
	if err := stub.Send(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte("k"),
		Value:          []byte("v"),
	}); err != nil {
		t.Fatalf("stub send: %v", err)
	}
	if len(stub.calls) != 1 || *stub.calls[0].TopicPartition.Topic != "t" {
		t.Fatalf("stub did not record call: %+v", stub.calls)
	}
}

// ---- kafka ExportSpans coverage ----

// recordingSpan makes a single span with the given name/attributes and
// returns the recorded ReadOnlySpan ready to feed into ExportSpans.
func recordingSpan(t *testing.T, name string, attrs ...attribute.KeyValue) sdktrace.ReadOnlySpan {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), name, trace.WithAttributes(attrs...))
	span.End()
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 recorded span, got %d", len(spans))
	}
	return spans[0]
}

// TestKafkaTopicExporter_ExportSpans_WithSpans drives the inner loop:
// buildSpanRecord → json.Marshal → Send. The stub verifies the
// produced message shape (key=trace_id, value=valid JSON).
func TestKafkaTopicExporter_ExportSpans_WithSpans(t *testing.T) {
	s := recordingSpan(t, "kafka-span",
		attribute.String("bucket", "default"),
		attribute.Int("retries", 3),
	)
	stub := &stubKafkaSender{}
	exp := &kafkaTopicExporter{producer: stub, topic: "spans"}

	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{s}); err != nil {
		t.Fatalf("ExportSpans: %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 Send call, got %d", len(stub.calls))
	}
	if stub.calls[0].Key == nil || string(stub.calls[0].Key) == "" {
		t.Fatal("expected non-empty key (trace_id)")
	}
	if !contains(stub.calls[0].Value, `"name":"kafka-span"`) {
		t.Fatalf("expected payload to contain span name, got: %s", stub.calls[0].Value)
	}
	if !contains(stub.calls[0].Value, `"bucket":"default"`) {
		t.Fatalf("expected payload to contain flattened string attribute, got: %s", stub.calls[0].Value)
	}
	if !contains(stub.calls[0].Value, `"retries":"3"`) {
		t.Fatalf("expected payload to contain flattened int attribute, got: %s", stub.calls[0].Value)
	}
}

// TestKafkaTopicExporter_ExportSpans_PropagatesError covers the
// "Send returns error" branch — the wrapper must prefix "produce".
func TestKafkaTopicExporter_ExportSpans_PropagatesError(t *testing.T) {
	s := recordingSpan(t, "boom")
	stub := &stubKafkaSender{err: errBrokerUnreachable}
	exp := &kafkaTopicExporter{producer: stub, topic: "x"}

	err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{s})
	if err == nil {
		t.Fatal("expected Send failure to surface")
	}
	if !contains([]byte(err.Error()), "tracing: kafka_topic: produce") {
		t.Fatalf("error should be wrapped with produce prefix, got: %v", err)
	}
}

// errBrokerUnreachable is a sentinel used by stub tests. Defined as
// a typed error so error-wrapping assertions can compare to a stable
// string without depending on a real broker error.
var errBrokerUnreachable = errBroker("broker unreachable")

type errBroker string

func (e errBroker) Error() string { return string(e) }

// TestKafkaTopicExporter_ExportSpans_ContextCancelled drives the
// ctx.Done() branch by passing an already-cancelled context.
func TestKafkaTopicExporter_ExportSpans_ContextCancelled(t *testing.T) {
	s := recordingSpan(t, "cancelled")
	stub := &stubKafkaSender{}
	exp := &kafkaTopicExporter{producer: stub, topic: "x"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := exp.ExportSpans(ctx, []sdktrace.ReadOnlySpan{s})
	if err == nil {
		t.Fatal("expected ctx.Err() to surface")
	}
	if !contains([]byte(err.Error()), "context canceled") {
		t.Fatalf("error should wrap context canceled, got: %v", err)
	}
	if len(stub.calls) != 0 {
		t.Fatalf("Send should not have been called, got %d calls", len(stub.calls))
	}
}

// TestKafkaTopicExporter_Shutdown_FlushPending covers the partial-flush
// error path. When Flush reports still-pending messages, Shutdown
// returns an error after best-effort Close.
func TestKafkaTopicExporter_Shutdown_FlushPending(t *testing.T) {
	stub := &stubKafkaSender{}
	stub.SetFlushPending(7)
	exp := &kafkaTopicExporter{producer: stub, topic: "x"}

	err := exp.Shutdown(context.Background())
	if err == nil {
		t.Fatal("expected flush-pending error")
	}
	if !contains([]byte(err.Error()), "7 messages still pending") {
		t.Fatalf("error should report pending count, got: %v", err)
	}
}

// _ keeps imports referenced even if every kafka test is gated off
// in a future refactor.
var _ = time.Second