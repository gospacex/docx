package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gospacex/hubx/cache/docx/config"
)

// flushTimeoutMs is the librdkafka Flush wait used on exporter shutdown.
const flushTimeoutMs = 5000

// kafkaTopicExporter is a self-contained OTel SpanExporter that serialises
// each ReadOnlySpan as JSON and produces it to a Kafka topic.
//
// It owns the underlying *kafka.Producer and flushes+closes it on Shutdown.
type kafkaTopicExporter struct {
	producer kafkaSender
	topic    string
	shutdown bool
}

// spanRecord is the on-the-wire payload. It deliberately omits every
// ReadOnlySpan field that's not needed for trace_id assertion downstream
// (Attributes flattened to string, Status/Kind dropped). Bumping the
// payload shape is a breaking change for downstream consumers.
type spanRecord struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Name       string            `json:"name"`
	StartTime  string            `json:"start_time"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// buildSpanRecord projects a ReadOnlySpan onto the on-the-wire spanRecord.
// Extracted as a free function so the projection is unit-testable without
// a Kafka broker — the only branch (with-attrs vs without-attrs) is here.
func buildSpanRecord(s sdktrace.ReadOnlySpan) spanRecord {
	rec := spanRecord{
		TraceID:   s.SpanContext().TraceID().String(),
		SpanID:    s.SpanContext().SpanID().String(),
		Name:      s.Name(),
		StartTime: s.StartTime().String(),
	}
	if attrs := s.Attributes(); len(attrs) != 0 {
		rec.Attributes = make(map[string]string, len(attrs))
		for _, kv := range attrs {
			rec.Attributes[string(kv.Key)] = kv.Value.Emit()
		}
	}
	return rec
}

// kafkaSender is the minimal contract a Kafka producer must satisfy to
// plug into kafkaTopicExporter. It exists so unit tests can verify the
// per-span message shape, the Flush timeout branch, and the Close path
// without spinning up librdkafka. In production *kafka.Producer
// satisfies it via the librdkafkaProducerAdapter.
type kafkaSender interface {
	Send(msg *kafka.Message) error
	Flush(timeoutMs int) int
	Close()
}

// send adapts *kafka.Producer to kafkaSender. The real Produce call is
// non-blocking — librdkafka queues the message and reports delivery via
// the Events channel — so unit tests can call Send with an unreachable
// broker and only observe a successful queue admission.
func (e *kafkaTopicExporter) send(msg *kafka.Message) error {
	return e.producer.Send(msg)
}

// ExportSpans serialises each span and produces it to the topic with the
// trace_id as the partition key. Delivery is flushed during Shutdown by the
// surrounding BatchSpanProcessor / exporter lifecycle rather than per batch.
func (e *kafkaTopicExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}
	topic := e.topic
	for _, s := range spans {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rec := buildSpanRecord(s)
		payload, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("tracing: kafka_topic: marshal span: %w", err)
		}
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte(rec.TraceID),
			Value:          payload,
		}
		if err := e.send(msg); err != nil {
			return fmt.Errorf("tracing: kafka_topic: produce: %w", err)
		}
	}
	return nil
}

// Shutdown flushes pending messages and closes the owned producer.
// Idempotent: a second call is a no-op.
func (e *kafkaTopicExporter) Shutdown(ctx context.Context) error {
	if e.shutdown {
		return nil
	}
	e.shutdown = true
	if remaining := e.producer.Flush(flushTimeoutMs); remaining > 0 {
		// best-effort: still try to close, but report partial flush.
		e.producer.Close()
		return fmt.Errorf("tracing: kafka_topic: shutdown flush, %d messages still pending", remaining)
	}
	// 2s budget for Close so a stuck broker doesn't pin Shutdown.
	closed := make(chan struct{})
	go func() {
		e.producer.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		return fmt.Errorf("tracing: kafka_topic: producer close timed out")
	}
	return nil
}

// newKafkaTopicExporter builds a kafka.Producer directly from cfg (no
// inject mode) and wraps it in our local SpanExporter. librdkafka config
// keys are kept as inline literals to avoid carrying a wrapping struct.
func newKafkaTopicExporter(cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	cm := &kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(cfg.Addrs, ","),
		"acks":               cfg.Producer.Acks,
		"enable.idempotence": cfg.Producer.Idempotent,
	}
	if cfg.Producer.LingerMs > 0 {
		_ = cm.SetKey("linger.ms", cfg.Producer.LingerMs)
	}
	if cfg.Kafka.SecurityProtocol != "" {
		_ = cm.SetKey("security.protocol", cfg.Kafka.SecurityProtocol)
	}
	if cfg.Auth.Username != "" {
		mech := cfg.Kafka.SASLMechanism
		if mech == "" {
			mech = "PLAIN"
		}
		_ = cm.SetKey("sasl.mechanisms", mech)
		_ = cm.SetKey("sasl.username", cfg.Auth.Username)
		_ = cm.SetKey("sasl.password", cfg.Auth.Password)
	}

	producer, err := kafka.NewProducer(cm)
	if err != nil {
		return nil, fmt.Errorf("tracing: kafka_topic: new producer: %w", err)
	}
	return &kafkaTopicExporter{producer: &librdkafkaProducerAdapter{p: producer}, topic: cfg.Producer.Topic}, nil
}

// librdkafkaProducerAdapter bridges *kafka.Producer to the kafkaSender
// interface so ExportSpans can be exercised with a stub sender in unit
// tests without spinning up librdkafka. The adapter is intentionally
// thin — it forwards to Produce verbatim.
type librdkafkaProducerAdapter struct {
	p *kafka.Producer
}

func (a *librdkafkaProducerAdapter) Send(msg *kafka.Message) error {
	return a.p.Produce(msg, nil)
}

func (a *librdkafkaProducerAdapter) Flush(timeoutMs int) int {
	return a.p.Flush(timeoutMs)
}

func (a *librdkafkaProducerAdapter) Close() {
	a.p.Close()
}
