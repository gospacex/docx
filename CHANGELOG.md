# Changelog

All notable changes to the **docx** cache SDK (root + `couchbase/` + `mongo/` sub-modules)
are documented in this file. The three modules share a single version line
(`hubx/cache/docx`, `hubx/cache/couchbase`, `hubx/cache/mongo`) so consumers
can pin one string across the monorepo.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Fixed
- README badge, `couchbase.Version`, and `mongo.Version` are now unified at
  `0.2.0-dev` (previously drifted to `2.0.0` in the sub-modules).
- `Config.ComputeContentHash` removed (zero callers; identical to `ContentHash`).

---

## [0.2.0-dev] — 2026-06-14

### Changed
- **TracingConfig field rename (breaking).** The legacy flat vocabulary was
  replaced with the nested `tracing:` block now shared by `docx`,
  `couchbase`, and `mongo`. See [`MIGRATION.md`](MIGRATION.md) for the
  field-by-field mapping. (`config/tracing.go:13-24`)
- `Backend` → `Exporter` (values: `jaeger` | `kafka_topic` | `redis_stream`).
- `Mode` removed — sampling now lives entirely in the OTel SDK and is
  configured via `SamplerType` / `SamplerRatio`.
- `KafkaBrokers` → `Addrs`. `KafkaTopic` → `Producer.Topic`.
  `KafkaSASLMechanism` → `Kafka.SASLMechanism`.
  `KafkaSASLUsername` / `KafkaSASLPassword` → `Auth.Username` / `Auth.Password`.
- `RedisAddr` → `Addrs[0]`. `RedisUsername` / `RedisPassword` →
  `Redis.Username` / `Redis.Password` (with `Auth` fallback).
- `Stream` → `Producer.Topic` (shared between Kafka and Redis exporters).
- `KafkaProducer` / `RedisClient` inject modes removed — the SDK now builds
  its own clients from YAML.

### Added
- `observability.WithNoop()` — install a TracerProvider that drops every span,
  for unit tests and CLI tools that want OTel API compatibility without a
  collector. (`observability/init.go:23`)
- `Config.CacheFingerprint()` returns the canonical cache key derived from
  the configuration; cluster keys strip the `Bucket` / `Database` /
  `Collection` fields so siblings share a cluster.
- `Config.ContentHash()` for cheap equality checks (errors swallowed).
- Error cooldown on the connection singleton: 30s after a build failure
  the cached error is reused without a retry. (`couchbase/instance.go:16`,
  `mongo/instance.go:16`)
- `kafka_topic` exporter flushes pending messages on `Shutdown` with a 5s
  librdkafka timeout and a 2s `Close` budget so a stuck broker cannot
  pin `Shutdown`.
- Layered dependency boundary enforced by CI:
  `scripts/check_deps.sh` forbids `observability/tracing` from importing
  the `mqx` family or the `couchbase` / `mongo` sub-modules.

### Removed
- `Backend` / `Mode` / `KafkaBrokers` / `KafkaTopic` / `KafkaSASL*` /
  `RedisAddr` / `RedisUsername` / `RedisPassword` / `Stream` fields on
  `TracingConfig`. The corresponding `tracing/tracing.go` constants were
  also renamed (`BackendJaeger` → `ExporterJaeger`, etc.).
- `Config.ComputeContentHash()` — kept as `Config.ContentHash()`.

---

## [0.1.0] — 2026-06-08

### Added
- Initial release of the `docx` public layer:
  - `config.TracingConfig` with Jaeger / Kafka-topic / Redis-stream exporters.
  - `observability.InitTracing` / `ShutdownTracing` / `StartSpan` /
    `InjectTrace` / `ExtractTrace` / `SetBaggage` / `GetBaggage`.
  - `utils.ExpandEnvVars` (`${env:NAME}` / `${env:NAME:-default}` syntax)
    and `utils.Fingerprint` (sha256 of JSON-marshalled config).
- `couchbase` sub-module: `COS` / `COC` / `CPS` / `CPC` entry points,
  `Bucket.Get` / `Insert` / `Upsert` / `Remove` / `Ping` / `HealthCheck`,
  traced wrappers (`GetTrace` / `InsertTrace` / `UpdateTrace` / `DeleteTrace`).
- `mongo` sub-module: `MOC` / `MOS` / `MPC` / `MPS` entry points,
  `Collection.Find` / `FindOne` / `InsertOne` / `UpdateOne` / `DeleteOne`,
  traced wrappers (`FindTrace` / `FindOneTrace` / `InsertTrace` /
  `UpdateTrace` / `DeleteTrace`).
- Five runnable examples under `example/`: Jaeger, Kafka topic, Redis stream,
  noop, and MongoDB.

[Unreleased]: https://github.com/gospacex/hubx/cache/docx/compare/v0.2.0-dev...HEAD
[0.2.0-dev]: https://github.com/gospacex/hubx/cache/docx/compare/v0.1.0...v0.2.0-dev
[0.1.0]: https://github.com/gospacex/hubx/cache/docx/releases/tag/v0.1.0