# docx — Public Layer Cache Library

[![Go Version](https://img.shields.io/badge/go-1.26.2-blue)](https://go.dev)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44.0-purple)](https://opentelemetry.io)
[![SemVer](https://img.shields.io/badge/version-0.2.0--dev-yellow)](CHANGELOG.md)

**docx** is a Go library that provides a consistent, opinionated abstraction layer over Couchbase and MongoDB, with pluggable distributed tracing via OpenTelemetry.

It wraps the underlying database drivers with connection singleton caching, explicit tracing initialization, YAML-based configuration, and self-contained span exporters — so your application code talks to one library instead of juggling three SDKs.

---

## Architecture

docx is a **multi-module monorepo** with three Go modules:

```
docx/                         # github.com/gospacex/hubx/cache/docx
├── config/                   # TracingConfig, CouchbaseConfig, MongoConfig
├── observability/            # InitTracing, ShutdownTracing, StartSpan, context propagation
│   └── tracing/              # SpanExporter: Jaeger / Kafka Topic / Redis Stream
├── utils/                    # Config fingerprinting, env-var expansion
├── test/                     # Shared test helpers
├── couchbase/                # github.com/gospacex/hubx/cache/couchbase (sub-module)
└── mongo/                    # github.com/gospacex/hubx/cache/mongo (sub-module)
```

| Module | Import path | Purpose |
|---|---|---|
| Root | `github.com/gospacex/hubx/cache/docx` | Config, tracing, utilities |
| Couchbase | `github.com/gospacex/hubx/cache/couchbase` | Couchbase connection + CRUD |
| MongoDB | `github.com/gospacex/hubx/cache/mongo` | MongoDB connection + CRUD |

---

## Features

- **Connection singleton caching** — `sync.Map`-based `getOrCreate` with 30s error cooldown; reconnecting with the same config reuses the existing connection.
- **Distributed tracing** — traced helpers emit OpenTelemetry spans after the application explicitly initializes tracing once.
- **Three span export modes** (self-contained, no external dependency injection):
  - **Jaeger** — OTLP gRPC or HTTP to a Jaeger collector.
  - **Kafka topic** — serialises spans as JSON to a Kafka topic via `confluent-kafka-go/v2`.
  - **Redis stream** — `XAdd`s spans to a Redis Stream via `go-redis/v9`.
- **YAML-based configuration** — one config file per data source, with `${env:VAR:-default}` expansion.
- **Context propagation** — `InjectTrace`/`ExtractTrace` helpers for propagating trace context across service boundaries.
- **Layered dependency boundary** — `observability/tracing` is forbidden from importing any sub-module or mqx package, enforced by CI (`scripts/check_deps.sh`).
- **Go 1.26.2** — built for the latest Go toolchain.

---

## Getting Started

### Prerequisites

- Go 1.26.2 or later

### Installation

```bash
go get github.com/gospacex/hubx/cache/docx
```

For the Couchbase or Mongo sub-modules:

```bash
go get github.com/gospacex/hubx/cache/couchbase
go get github.com/gospacex/hubx/cache/mongo
```

### Minimal example

```go
package main

import (
    "context"
    "log"

    "github.com/gospacex/hubx/cache/couchbase"
)

func main() {
    ctx := context.Background()
    bucket, err := couchbase.CPS(ctx, "couchbase.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer bucket.Close()

    // Typed CRUD
    if _, err := bucket.Upsert("my-key", map[string]string{"hello": "world"}); err != nil {
        log.Fatal(err)
    }
}
```

```yaml
# couchbase.yaml
endpoints:
  - localhost:8091
bucket: my-bucket
username: admin
password: pass
```

### Enabling tracing explicitly

`COS` / `COC` / `MOS` / `MOC` no longer install a global tracer provider
implicitly. If you want traced helpers such as `GetTrace`, `InsertTrace`,
or `FindTrace`, initialize tracing once in your application and shut it
back down on exit:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/gospacex/hubx/cache/couchbase"
    "github.com/gospacex/hubx/cache/docx/observability"
)

func main() {
    ctx := context.Background()

    raw, err := os.ReadFile("couchbase.yaml")
    if err != nil {
        log.Fatal(err)
    }
    cfg, err := couchbase.ParseConfig(raw)
    if err != nil {
        log.Fatal(err)
    }
    if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
        log.Fatal(err)
    }
    defer observability.ShutdownTracing(ctx)

    bucket, err := couchbase.COS(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer bucket.Close()
}
```

The `tracing:` block in YAML is still the canonical source for exporter,
sampling, and auth settings; it is simply consumed explicitly by the
application now.

---

## Configuration

### TracingConfig

The tracing configuration is the heart of docx's observability. It is a nested YAML struct consumed by both the root module and the sub-modules.

```yaml
tracing:
  enabled: true                                # opt-in
  service_name: my-service                     # required when enabled
  exporter: jaeger                             # jaeger | kafka_topic | redis_stream
  protocol: grpc                               # grpc | http (jaeger only)
  endpoint: localhost:4317                     # OTLP collector endpoint
  insecure: true                               # skip TLS
  sampler_type: parentbased_traceidratio
  sampler_ratio: 0.1
  addrs: [broker1:9092]                        # kafka / redis broker addresses
  auth:
    username: user
    password: ${env:TRACING_PASS}
    mechanism: SCRAM-SHA-512
  producer:
    topic: my-traces
    acks: all                                  # default (kafka_topic only)
    idempotent: true                           # default (kafka_topic only)
    linger_ms: 5
  kafka:
    security_protocol: SASL_PLAINTEXT
    sasl_mechanism: SCRAM-SHA-512
  redis:
    db: 0
    pool_size: 10
```

Validation rules are documented in [`config/tracing.go`](config/tracing.go) — the `Validate()` method is called automatically by `InitTracing`.

### Sub-module configs

```yaml
# Couchbase
endpoints:
  - localhost:8091
username: admin
password: ${env:CB_PASS}
bucket: my-bucket
tracing: { ... }
```

```yaml
# MongoDB
uri: mongodb://localhost:27017
username: admin
password: ${env:MONGO_PASS}
database: my-db
collection: my-coll
tracing: { ... }
```

---

## API Overview

### Root module (`docx`)

| Package | Key functions |
|---|---|
| `config` | `TracingConfig`, `CouchbaseConfig`, `MongoConfig` structs; `Validate()` |
| `observability` | `InitTracing(ctx, cfg)`, `ShutdownTracing(ctx)`, `StartSpan(ctx, name)`, `SetBaggage`, `GetBaggage`, `InjectTrace`, `ExtractTrace` |
| `observability/tracing` | `NewExporter(cfg)` — factory returning `sdktrace.SpanExporter` |
| `utils` | `Fingerprint(cfg)`, `ExpandEnvVars(s)` |

### Couchbase sub-module

| Function | Shortcut for |
|---|---|
| `COS(ctx, cfg)` | Couchbase **O**pen **S**tandard — returns `*Bucket` |
| `COC(ctx, cfg)` | Couchbase **O**pen **C**luster — returns `*Cluster` |
| `CPS(ctx, path)` | Couchbase **P**arse-and-Connect **S**tandard — YAML → `*Bucket` |
| `CPC(ctx, path)` | Couchbase **P**arse-and-Connect **C**luster — YAML → `*Cluster` |

**Bucket methods**: `Get`, `Insert`, `Upsert`, `Remove`, `Ping`, `HealthCheck`, `Close()`

**Traced wrappers**: `GetTrace`, `InsertTrace`, `UpdateTrace`, `DeleteTrace` — same signatures with automatic span emission.

### Mongo sub-module

| Function | Shortcut for |
|---|---|
| `MOC(ctx, cfg)` | Mongo **O**pen **C**lient — returns `*Client` |
| `MOS(ctx, cfg)` | Mongo **O**pen **S**tandard — returns `*Collection` |
| `MPC(ctx, path)` | Mongo **P**arse-and-Connect **C**lient — YAML → `*Client` |
| `MPS(ctx, path)` | Mongo **P**arse-and-Connect **S**tandard — YAML → `*Collection` |

**Collection methods**: `Find`, `FindOne`, `InsertOne`, `UpdateOne`, `DeleteOne`, `HealthCheck`, `Close(ctx)`

**Traced wrappers**: `FindTrace`, `FindOneTrace`, `InsertTrace`, `UpdateTrace`, `DeleteTrace`

---

## Examples

Runnable end-to-end examples live under [`example/`](example/):

| Example | Description |
|---|---|
| [`01-jaeger`](example/01-jaeger) | Couchbase CRUD + Jaeger OTLP/gRPC exporter |
| [`02-kafka-topic`](example/02-kafka-topic) | Couchbase CRUD + Kafka topic exporter |
| [`03-redis-stream`](example/03-redis-stream) | Couchbase CRUD + Redis stream exporter |
| [`04-noop`](example/04-noop) | Couchbase CRUD + `WithNoop()` tracing |
| [`mongo_test`](example/mongo_test) | MongoDB CRUD + Jaeger OTLP/gRPC exporter |

```bash
cd example/01-jaeger && go run .
```

---

## Build & Test

All commands are in the top-level [`Makefile`](Makefile).

```bash
make build          # Build root module
make build-all      # Build root + sub-modules
make test           # Run unit tests (short mode)
make test-all       # Run all unit tests across modules
make test-race      # Run with -race detector
make cover          # Coverage report
make cover-html     # Interactive HTML coverage
make vet            # go vet ./...
make deps-check     # Enforce layered dependency boundaries
make ci             # Full CI gate: deps → vet → openspec → test → coverage
```

---

## Dependencies

| Library | Purpose |
|---|---|
| `go.opentelemetry.io/otel` | OpenTelemetry tracing SDK |
| `confluentinc/confluent-kafka-go/v2` | Kafka producer (Kafka topic exporter) |
| `redis/go-redis/v9` | Redis client (Redis stream exporter) |
| `couchbase/gocb/v2` | Couchbase Go SDK (couchbase sub-module only) |
| `go.mongodb.org/mongo-driver` | MongoDB Go driver (mongo sub-module only) |
| `gopkg.in/yaml.v3` | YAML parsing |
| `google.golang.org/grpc` | OTLP/gRPC transport |

---

## Migration

If you are upgrading from an older version that used the legacy `backend`/`mode`/`kafka_*` flat vocabulary, see [`MIGRATION.md`](MIGRATION.md) for the complete field-by-field mapping table.

---

## Contributing

1. Run `make ci` before pushing — it gates dependency boundaries, code quality, and test coverage.
2. Keep the layered architecture: `observability/tracing` must not import `mqx` or any sub-module.
3. Follow existing patterns for config structs, error sentinels, and singleton caching.

---

## License

Internal — Space X HubX.
