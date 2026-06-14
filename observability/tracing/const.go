package tracing

// Exporter type constants. These match config.TracingConfig.Exporter values
// and the values used in OTel exporter choice. Renamed from Backend* to
// Exporter* on 2026-06-13 to align with the new TracingConfig vocabulary.
const (
	ExporterJaeger      = "jaeger"
	ExporterKafkaTopic  = "kafka_topic"
	ExporterRedisStream = "redis_stream"
)

// Protocol values accepted by TracingConfig.Protocol. Default is grpc.
const (
	ProtocolGRPC = "grpc"
	ProtocolHTTP = "http"
)
