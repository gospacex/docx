# Changelog

All notable changes to docx are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking — `config.TracingConfig` field renames

The tracing configuration vocabulary has been realigned with the mqx
configuration tree (`kafkax` / `redisx` / `observability`). YAML files
written against the old vocabulary MUST be migrated; see
[MIGRATION.md](MIGRATION.md) for the full mapping table.

| Old field            | New field                                              |
|----------------------|--------------------------------------------------------|
| `backend`            | `exporter` (`jaeger` / `kafka_topic` / `redis_stream`) |
| `mode`               | _removed_ — sampling lives in OTel SDK                 |
| `kafka_brokers`      | `addrs` (shared between kafka and redis)               |
| `kafka_topic`        | `producer.topic`                                       |
| `kafka_sasl_mechanism` | `kafka.sasl_mechanism`                               |
| `kafka_sasl_username` / `kafka_sasl_password` | `auth.username` / `auth.password` |
| `redis_addr`         | `addrs[0]`                                             |
| `redis_username` / `redis_password` | `redis.username` / `redis.password` (with `auth` fallback) |
| `stream`             | `producer.topic` (shared)                              |
| `kafka_producer`     | _removed_ — direct injection no longer supported       |
| `redis_client`       | _removed_ — direct injection no longer supported       |

### Added — `config.TracingConfig` new fields

- `protocol` (`grpc` / `http`; default `grpc`) — selects the OTLP
  transport for the `jaeger` exporter.
- `auth` (`TracingAuthConfig`) — username/password/mechanism/token.
- `producer` (`TracingProducerConfig`) — topic/acks/idempotent/linger_ms.
- `kafka` (`TracingKafkaConfig`) — security_protocol/sasl_mechanism.
- `redis` (`TracingRedisConfig`) — db/username/password/pool_size.

`Validate()` now defaults `producer.acks = "all"` and forces
`producer.idempotent = true` for the `kafka_topic` exporter (matches
mqx `kafkax.POS` defaults).

### Removed — `mqx` dependency in the public layer

The `observability/tracing` package no longer imports any `github.com/gospacex/mqx/...`
subpackage. Both the Kafka topic and the Redis stream exporter are
self-built on top of `confluent-kafka-go/v2` and `go-redis/v9`,
respectively. Top-level `go.mod` has dropped the `replace` and the
`require` for mqx.

`scripts/check_deps.sh` enforces this rule: any `*.go` under
`observability/tracing` that imports `mqx/kafkax`, `mqx/redisx`, or
`mqx/observability/exporter` fails the CI gate.

### Added — observability/tracing constants renamed

`BackendJaeger` / `BackendKafkaTopic` / `BackendRedisStream` → `ExporterJaeger`
/ `ExporterKafkaTopic` / `ExporterRedisStream`. New `ProtocolGRPC` /
`ProtocolHTTP` for OTLP transport selection.

### Examples

Four runnable end-to-end examples under `example/`:

- `01-jaeger/` — Couchbase + OTLP/gRPC to a Jaeger collector.
- `02-kafka-topic/` — Couchbase + spans serialised to a Kafka topic.
- `03-redis-stream/` — Couchbase + spans XAdd-ed to a Redis Stream.
- `04-noop/` — Couchbase + `WithNoop()` for unit tests / local dev.

Each example is its own Go module with a `couchbase.yaml` and a
`README.md`. The build system targets live in the top-level `Makefile`.

### Testing

Public-layer unit tests added:

- `config/tracing_test.go` — 14 `Validate()` scenarios, 100% coverage.
- `utils/env_test.go` + `fingerprint_test.go` — 6 scenarios, 94.1%.
- `observability/init_test.go` — noop path + validation error
  propagation, 37.1% (rest is integration-tested via examples).

## [0.1.0] — 2026-06-01

Initial release with the legacy `Backend` field and the inject-mode
support for externally-built Kafka producers / Redis clients.
