package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gospacex/hubx/cache/docx/config"
)

// redisStreamExporter is a self-contained OTel SpanExporter that XAdds each
// ReadOnlySpan as a single entry to a Redis Stream. It owns the underlying
// *redis.Client and closes it on Shutdown.
type redisStreamExporter struct {
	client   *redis.Client
	stream   string
	shutdown bool
}

type redisSpanRecord struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Name       string            `json:"name"`
	StartTime  string            `json:"start_time"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ExportSpans XAdds each span as one stream entry with the JSON payload in
// the "span" field. Trace_id is not used as a field key directly (consumers
// can decode it from the payload), but the entry's natural ordering gives
// stream-side scans a stable iteration order.
func (e *redisStreamExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}
	for _, s := range spans {
		rec := redisSpanRecord{
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
		payload, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("tracing: redis_stream: marshal span: %w", err)
		}
		if err := e.client.XAdd(ctx, &redis.XAddArgs{
			Stream: e.stream,
			Values: map[string]interface{}{"span": payload},
		}).Err(); err != nil {
			return fmt.Errorf("tracing: redis_stream: XAdd: %w", err)
		}
	}
	return nil
}

// Shutdown closes the owned redis client. Idempotent.
func (e *redisStreamExporter) Shutdown(ctx context.Context) error {
	_ = ctx
	if e.shutdown {
		return nil
	}
	e.shutdown = true
	return e.client.Close()
}

// newRedisStreamExporter builds a *redis.Client directly from cfg (no
// inject mode). Auth falls back from Redis.Username/Password to
// Auth.Username/Password so the YAML can stay terse.
func newRedisStreamExporter(cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	username := cfg.Redis.Username
	if username == "" {
		username = cfg.Auth.Username
	}
	password := cfg.Redis.Password
	if password == "" {
		password = cfg.Auth.Password
	}
	opts := &redis.Options{
		Addr:     cfg.Addrs[0],
		Username: username,
		Password: password,
		DB:       cfg.Redis.DB,
	}
	if cfg.Redis.PoolSize > 0 {
		opts.PoolSize = cfg.Redis.PoolSize
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("tracing: redis_stream: ping: %w", err)
	}

	return &redisStreamExporter{client: client, stream: cfg.Producer.Topic}, nil
}
