# Migration Guide — docx tracing field rename

The `docx-trace-align-mqx-style` change re-aligned
`config.TracingConfig` with the mqx configuration tree
(`kafkax` / `redisx` / `observability`). This document is the
authoritative field-by-field mapping for upgrading existing
configuration files and Go callers.

> TL;DR — every `kafka_*` field collapsed into the mqx-style nested
> `auth` / `kafka` / `producer` substruct; the `backend` field became
> `exporter`; the `kafka_producer` / `redis_client` injection knobs
> disappeared; `producer.topic` is now the shared name for both Kafka
> topics and Redis stream names.

## YAML diff

### Before (legacy)

```yaml
tracing:
  enabled: true
  service_name: orders
  backend: kafka_topic
  mode: always_on
  sampler_type: parentbased_traceidratio
  sampler_ratio: 0.1
  insecure: true
  kafka_brokers: [broker1:9092, broker2:9092]
  kafka_topic: orders-traces
  kafka_sasl_mechanism: SCRAM-SHA-512
  kafka_sasl_username: orders
  kafka_sasl_password: ${env:ORDERS_KAFKA_PASS}
```

### After (mqx-aligned)

```yaml
tracing:
  enabled: true
  service_name: orders
  exporter: kafka_topic
  protocol: grpc            # only consulted by exporter=jaeger
  sampler_type: parentbased_traceidratio
  sampler_ratio: 0.1
  insecure: true
  addrs: [broker1:9092, broker2:9092]
  auth:
    username: orders
    password: ${env:ORDERS_KAFKA_PASS}
    mechanism: SCRAM-SHA-512
  kafka:
    security_protocol: SASL_PLAINTEXT
    sasl_mechanism: SCRAM-SHA-512
  producer:
    topic: orders-traces
    acks: all               # default applied by Validate()
    idempotent: true        # default applied by Validate()
```

## Field mapping

| Legacy field                                | New field                                                | Notes |
|---------------------------------------------|----------------------------------------------------------|-------|
| `backend`                                   | `exporter`                                               | values unchanged: `jaeger` / `kafka_topic` / `redis_stream` |
| `mode`                                      | _removed_                                                | sampling now flows through `sampler_type` / `sampler_ratio`; if you were using `mode=always_on`, set `sampler_ratio: 1.0` |
| `kafka_brokers`                             | `addrs`                                                  | renamed; the slice is shared with redis |
| `kafka_topic`                               | `producer.topic`                                         | the topic name; `producer.acks` / `producer.idempotent` default to mqx values |
| `kafka_sasl_mechanism`                      | `kafka.sasl_mechanism`                                   | nested under `kafka:` |
| `kafka_sasl_username`                       | `auth.username`                                          | nested under `auth:` |
| `kafka_sasl_password`                       | `auth.password`                                          | nested under `auth:` |
| `redis_addr`                                | `addrs[0]`                                               | docx only uses the first address for redis |
| `redis_username`                            | `redis.username` (with `auth.username` fallback)         | go-redis expects creds in `redis:`, but if you already use `auth:`, leave `redis:` empty |
| `redis_password`                            | `redis.password` (with `auth.password` fallback)         | same |
| `stream`                                    | `producer.topic`                                         | shared between kafka and redis |
| `kafka_producer`                            | _removed_                                                | direct injection no longer supported — use `addrs` / `auth` / `kafka` / `producer` |
| `redis_client`                              | _removed_                                                | direct injection no longer supported — use `addrs` / `auth` / `redis` |
| _new_ `protocol`                            | `protocol` (`grpc` / `http`)                             | selects OTLP transport for `exporter=jaeger`; default `grpc` |
| _new_ `auth.mechanism`                      | `auth.mechanism`                                         | SASL mechanism for kafka; mqx's `AuthConfig.Mechanism` |
| _new_ `auth.token`                          | `auth.token`                                             | reserved for token-based kafka/redis auth |
| _new_ `kafka.security_protocol`             | `kafka.security_protocol`                                | `PLAINTEXT` / `SASL_PLAINTEXT` / `SASL_SSL` |
| _new_ `redis.db` / `redis.pool_size`        | `redis.db` / `redis.pool_size`                           | go-redis options |
| _new_ `producer.linger_ms`                  | `producer.linger_ms`                                     | forwarded to librdkafka |

## Go struct changes

```go
// Legacy
type TracingConfig struct {
    Backend             string   `yaml:"backend"`
    Mode                string   `yaml:"mode"`
    KafkaBrokers        []string `yaml:"kafka_brokers"`
    KafkaTopic          string   `yaml:"kafka_topic"`
    KafkaSASLMechanism  string   `yaml:"kafka_sasl_mechanism"`
    KafkaSASLUsername   string   `yaml:"kafka_sasl_username"`
    KafkaSASLPassword   string   `yaml:"kafka_sasl_password"`
    RedisAddr           string   `yaml:"redis_addr"`
    RedisUsername       string   `yaml:"redis_username"`
    RedisPassword       string   `yaml:"redis_password"`
    Stream              string   `yaml:"stream"`
    KafkaProducer       any      `yaml:"-"`
    RedisClient         any      `yaml:"-"`
    /* …unchanged: ServiceName, Endpoint, Sampler*, Insecure, Headers … */
}

// New
type TracingConfig struct {
    Exporter    string                `yaml:"exporter"`
    Protocol    string                `yaml:"protocol"`
    Addrs       []string              `yaml:"addrs"`
    Auth        TracingAuthConfig     `yaml:"auth"`
    Producer    TracingProducerConfig `yaml:"producer"`
    Kafka       TracingKafkaConfig    `yaml:"kafka"`
    Redis       TracingRedisConfig    `yaml:"redis"`
    /* …unchanged: ServiceName, Endpoint, Sampler*, Insecure, Headers … */
}
```

## Removed APIs

The previous code allowed a caller to inject a pre-built
`*kafka.Producer` or `*redis.Client` via
`cfg.KafkaProducer` / `cfg.RedisClient`. The new code always builds the
client itself from `cfg.Addrs` / `cfg.Auth` / `cfg.Kafka` /
`cfg.Producer` (or the redis-side equivalents). This keeps the export
side fully self-contained: the public-layer module no longer imports
`github.com/gospacex/mqx/...` at all, and `scripts/check_deps.sh`
enforces the boundary in CI.

If you need a pre-built client (e.g. to share a connection pool with
another subsystem), wrap it behind your own `tracing.SpanExporter` and
hand the result to `otel.SetTracerProvider` directly — bypass
`InitTracing`.

## Validation rules (new)

`Validate()` runs the following rules in order; the first failure
short-circuits:

1. `Enabled == false` ⇒ no-op, no error.
2. `ServiceName == ""` ⇒ `service_name is required when tracing is enabled`.
3. `Protocol == ""` ⇒ defaulted to `"grpc"`.
4. `Exporter` switch:
   - `jaeger`: `Endpoint` must be non-empty; `Protocol` must be `grpc` or `http`.
   - `kafka_topic`: `Addrs` must be non-empty; `Producer.Topic` must be non-empty; `Producer.Acks` defaults to `"all"`; `Producer.Idempotent` forced to `true`.
   - `redis_stream`: `Addrs` must be non-empty; `Producer.Topic` must be non-empty (the stream name).
   - `""`: error `exporter is required when tracing is enabled`.
   - anything else: error `unknown exporter`.
5. `SamplerRatio` must lie in `[0, 1]`.

## Where the field names live

| Concern | Type |
|---------|------|
| Top-level tracing config | `github.com/gospacex/hubx/cache/docx/config.TracingConfig` |
| Auth | `config.TracingAuthConfig` |
| Kafka-specific | `config.TracingKafkaConfig` |
| Redis-specific | `config.TracingRedisConfig` |
| Producer (topic, acks, idempotent) | `config.TracingProducerConfig` |
| Exporter type constants | `tracing.ExporterJaeger` / `ExporterKafkaTopic` / `ExporterRedisStream` |
| Protocol constants | `tracing.ProtocolGRPC` / `ProtocolHTTP` |

The constants in the `tracing` package are the values you assign to
`config.TracingConfig.Exporter` (and `Protocol` for jaeger).

## Quick test

After updating your YAML, run:

```bash
go test ./config/...
go test ./observability/...
go test ./observability/tracing/...
```

The `config/tracing_test.go` cases cover all the new rules; if your
YAML passes, your code will too.
