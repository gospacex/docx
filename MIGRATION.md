# Migration Guide

This guide covers field renames between the legacy flat `TracingConfig`
vocabulary (used before `0.2.0-dev`) and the current nested `tracing:` block.

The change was applied on **2026-06-13** (commit `1d57fad`). Anything written
against the old fields will fail YAML parsing silently — the new struct
ignores unknown keys — so the only visible breakage is `Validate()` returning
`exporter is required when tracing is enabled` because the legacy `Backend`
field is no longer recognised.

## Top-level field renames

| Legacy field (pre-0.2.0-dev) | New field | Notes |
|---|---|---|
| `Backend` | `Exporter` | Values: `jaeger` / `kafka_topic` / `redis_stream` |
| `Mode` | _removed_ | Sampling now lives in `SamplerType` / `SamplerRatio` |
| `Protocol` | `Protocol` | Unchanged. Values: `grpc` / `http` (Jaeger only) |
| `Endpoint` | `Endpoint` | Unchanged |
| `ServiceName` | `ServiceName` | Unchanged |
| `SamplerType` | `SamplerType` | Unchanged |
| `SamplerRatio` | `SamplerRatio` | Unchanged |
| `Insecure` | `Insecure` | Unchanged |
| `Headers` | `Headers` | Unchanged |

## Kafka exporter renames

| Legacy | New |
|---|---|
| `KafkaBrokers` | `Addrs` |
| `KafkaTopic` | `Producer.Topic` |
| `KafkaSASLMechanism` | `Kafka.SASLMechanism` |
| `KafkaSASLUsername` | `Auth.Username` |
| `KafkaSASLPassword` | `Auth.Password` |
| `KafkaAcks` | `Producer.Acks` |
| `KafkaIdempotent` | `Producer.Idempotent` |
| `KafkaLingerMs` | `Producer.LingerMs` |
| `KafkaProducer` (inject mode) | _removed_ — SDK builds its own producer from YAML |

### Before (legacy)

```yaml
tracing:
  enabled: true
  backend: kafka_topic
  service_name: my-service
  kafka_brokers:
    - kafka-1:9092
  kafka_topic: otel-traces
  kafka_sasl_mechanism: SCRAM-SHA-512
  kafka_sasl_username: user
  kafka_sasl_password: ${env:TRACING_PASS}
```

### After (0.2.0-dev)

```yaml
tracing:
  enabled: true
  exporter: kafka_topic
  service_name: my-service
  addrs:
    - kafka-1:9092
  producer:
    topic: otel-traces
    acks: all
    idempotent: true
  auth:
    username: user
    password: ${env:TRACING_PASS}
    mechanism: SCRAM-SHA-512
  kafka:
    security_protocol: SASL_PLAINTEXT
    sasl_mechanism: SCRAM-SHA-512
```

## Redis exporter renames

| Legacy | New |
|---|---|
| `RedisAddr` | `Addrs[0]` (the first entry of the list) |
| `RedisUsername` | `Redis.Username` (falls back to `Auth.Username`) |
| `RedisPassword` | `Redis.Password` (falls back to `Auth.Password`) |
| `RedisDB` | `Redis.DB` |
| `RedisPoolSize` | `Redis.PoolSize` |
| `Stream` | `Producer.Topic` (shared with the Kafka exporter) |
| `RedisClient` (inject mode) | _removed_ — SDK builds its own client from YAML |

### Before (legacy)

```yaml
tracing:
  enabled: true
  backend: redis_stream
  service_name: my-service
  redis_addr: redis:6379
  redis_db: 0
  redis_username: user
  redis_password: ${env:REDIS_PASS}
  stream: otel-traces
```

### After (0.2.0-dev)

```yaml
tracing:
  enabled: true
  exporter: redis_stream
  service_name: my-service
  addrs:
    - redis:6379
  producer:
    topic: otel-traces
  auth:
    username: user
    password: ${env:REDIS_PASS}
  redis:
    db: 0
    username: user        # optional: falls back to auth.username
    password: ${env:REDIS_PASS}  # optional: falls back to auth.password
    pool_size: 10
```

## Jaeger exporter renames

| Legacy | New |
|---|---|
| `Backend: jaeger` | `Exporter: jaeger` |
| (no change to `Protocol`, `Endpoint`, `Insecure`, `Headers`) | — |

`Protocol` still drives the OTLP transport — `grpc` (default) or `http`.
`Auth.Username` / `Auth.Password` are materialised as a `Basic` header and
merged with `Headers["authorization"]` (password wins).

## Sampling changes (`Mode` removal)

The legacy `Mode` field had three values (`always`, `ratio`, `parentbased`)
plus a separate `Ratio` number. In `0.2.0-dev` sampling is configured
directly with `SamplerType` (one of `always_on`, `always_off`,
`traceidratio`, `parentbased_traceidratio`) and `SamplerRatio` in `[0, 1]`.

| Legacy | New |
|---|---|
| `mode: always` | `sampler_type: always_on` |
| `mode: never` | `sampler_type: always_off` |
| `mode: ratio` + `ratio: 0.1` | `sampler_type: traceidratio`, `sampler_ratio: 0.1` |
| `mode: parentbased` + `ratio: 0.1` | `sampler_type: parentbased_traceidratio`, `sampler_ratio: 0.1` |

If `SamplerType` is omitted, `Validate()` defaults it to
`parentbased_traceidratio` when `SamplerRatio ∈ (0, 1)`, otherwise to
`always_on`. This matches the legacy `Mode: always` default for ratio = 1.0.

## Constant renames (Go API)

The `tracing` package constants were renamed to match the new vocabulary:

| Legacy constant | New constant |
|---|---|
| `tracing.BackendJaeger` | `tracing.ExporterJaeger` |
| `tracing.BackendKafkaTopic` | `tracing.ExporterKafkaTopic` |
| `tracing.BackendRedisStream` | `tracing.ExporterRedisStream` |

`tracing.ProtocolGRPC` / `tracing.ProtocolHTTP` are unchanged.

## Behavioural changes

- `InitTracing` no longer installs a global tracer provider implicitly from
  `COS` / `COC` / `MOS` / `MOC`. Call `observability.InitTracing(ctx, cfg.Tracing)`
  once during application startup and `observability.ShutdownTracing(ctx)`
  on exit. The traced helpers (`GetTrace`, `InsertTrace`, etc.) now use the
  provider that the application installed, or the SDK's noop provider if
  none was installed.
- The `tracing:` block is still consumed explicitly from each sub-module's
  YAML; no automatic fall-through.
- `kafka_topic` exporter defaults `Producer.Acks` to `all` and
  `Producer.Idempotent` to `true` when they are empty (matches mqx parity).
- `redis_stream` exporter pings the broker on construction (3s timeout) so
  misconfiguration fails fast instead of at first span.