package tracing

import (
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gospacex/hubx/cache/docx/config"
)

// NewExporter dispatches to the per-exporter constructor based on
// cfg.Exporter. The returned SpanExporter OWNS its underlying client
// (kafka.Producer / redis.Client) and closes it on Shutdown.
func NewExporter(cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterJaeger:
		return newJaegerExporter(cfg)
	case ExporterKafkaTopic:
		return newKafkaTopicExporter(cfg)
	case ExporterRedisStream:
		return newRedisStreamExporter(cfg)
	default:
		return nil, fmt.Errorf("tracing: unknown exporter %q", cfg.Exporter)
	}
}
