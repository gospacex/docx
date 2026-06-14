# docx — 公共层缓存库

[![Go Version](https://img.shields.io/badge/go-1.26.2-blue)](https://go.dev)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44.0-purple)](https://opentelemetry.io)
[![SemVer](https://img.shields.io/badge/version-0.2.0--dev-yellow)](CHANGELOG.md)

**docx** 是一个 Go 库，为 Couchbase 和 MongoDB 提供统一的、约定优于配置的抽象层，并附带基于 OpenTelemetry 的可插拔分布式追踪能力。

它在底层数据库驱动之上封装了连接单例缓存、自动初始化追踪、YAML 驱动配置以及自包含的 Span 导出器——让你的业务代码只需对接这一个库，而无需同时处理三个 SDK。

---

## 架构

docx 是一个**多模块单体仓库**，包含三个 Go 模块：

```
docx/                         # github.com/gospacex/hubx/cache/docx
├── config/                   # TracingConfig, CouchbaseConfig, MongoConfig
├── observability/            # InitTracing, StartSpan, 链路传播
│   └── tracing/              # SpanExporter: Jaeger / Kafka Topic / Redis Stream
├── utils/                    # 配置指纹、环境变量展开
├── test/                     # 共享测试辅助
├── couchbase/                # github.com/gospacex/hubx/cache/couchbase (子模块)
└── mongo/                    # github.com/gospacex/hubx/cache/mongo (子模块)
```

| 模块 | 导入路径 | 职责 |
|---|---|---|
| 根模块 | `github.com/gospacex/hubx/cache/docx` | 配置、追踪、工具函数 |
| Couchbase | `github.com/gospacex/hubx/cache/couchbase` | Couchbase 连接 + CRUD |
| MongoDB | `github.com/gospacex/hubx/cache/mongo` | MongoDB 连接 + CRUD |

---

## 特性

- **连接单例缓存** — 基于 `sync.Map` + `sync.Mutex` 的 `getOrCreate`，带 30s 错误冷却；相同配置重复打开复用已有连接。
- **分布式追踪** — 每个数据库操作可自动生成 OpenTelemetry Span。
- **三种 Span 导出模式**（自包含，无需外部依赖注入）：
  - **Jaeger** — 通过 OTLP gRPC 或 HTTP 发送至 Jaeger Collector。
  - **Kafka Topic** — 将 Span 序列化为 JSON 发送至 Kafka Topic（基于 `confluent-kafka-go/v2`）。
  - **Redis Stream** — 将 Span 以 `XAdd` 写入 Redis Stream（基于 `go-redis/v9`）。
- **YAML 配置驱动** — 每个数据源一个配置文件，支持 `${env:VAR:-default}` 环境变量展开。
- **链路传播** — `InjectTrace` / `ExtractTrace` 辅助函数，用于跨服务边界传递追踪上下文。
- **分层依赖边界** — `observability/tracing` 禁止导入任何子模块或 mqx 包，由 CI 脚本 `scripts/check_deps.sh` 强制执行。
- **Go 1.26.2** — 基于最新 Go 工具链构建。

---

## 快速开始

### 前置条件

- Go 1.26.2 或更高版本

### 安装

```bash
go get github.com/gospacex/hubx/cache/docx
```

如需 Couchbase 或 Mongo 子模块：

```bash
go get github.com/gospacex/hubx/cache/couchbase
go get github.com/gospacex/hubx/cache/mongo
```

### 最小示例

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

    _ = bucket.Upsert(ctx, "my-key", map[string]string{"hello": "world"})
}
```

```yaml
# couchbase.yaml
address: localhost
username: admin
password: pass
bucket_name: my-bucket
tracing:
  enabled: true
  service_name: my-service
  exporter: jaeger
  endpoint: localhost:4317
```

---

## 配置

### TracingConfig

追踪配置是 docx 可观测性的核心，是一个嵌套的 YAML 结构体，同时被根模块和子模块消费。

```yaml
tracing:
  enabled: true                                # 启用追踪（opt-in）
  service_name: my-service                     # 启用时必填
  exporter: jaeger                             # jaeger | kafka_topic | redis_stream
  protocol: grpc                               # grpc | http（仅 jaeger）
  endpoint: localhost:4317                     # OTLP Collector 地址
  insecure: true                               # 跳过 TLS
  sampler_type: parentbased_traceidratio
  sampler_ratio: 0.1
  addrs: [broker1:9092]                        # kafka / redis 地址
  auth:
    username: user
    password: ${env:TRACING_PASS}
    mechanism: SCRAM-SHA-512
  producer:
    topic: my-traces
    acks: all                                  # 默认值（仅 kafka_topic）
    idempotent: true                           # 默认值（仅 kafka_topic）
    linger_ms: 5
  kafka:
    security_protocol: SASL_PLAINTEXT
    sasl_mechanism: SCRAM-SHA-512
  redis:
    db: 0
    pool_size: 10
```

验证规则详见 [`config/tracing.go`](config/tracing.go) — `Validate()` 方法由 `InitTracing` 自动调用。

### 子模块配置

```yaml
# Couchbase
address: localhost
username: admin
password: ${env:CB_PASS}
bucket_name: my-bucket
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

## API 总览

### 根模块（`docx`）

| 包 | 核心函数 / 类型 |
|---|---|
| `config` | `TracingConfig`、`CouchbaseConfig`、`MongoConfig`；`Validate()` |
| `observability` | `InitTracing(ctx, cfg)`、`StartSpan(ctx, name)`、`SetBaggage`、`GetBaggage`、`InjectTrace`、`ExtractTrace` |
| `observability/tracing` | `NewExporter(cfg)` — 工厂函数，返回 `sdktrace.SpanExporter` |
| `utils` | `Fingerprint(cfg)`、`ExpandEnvVars(s)` |

### Couchbase 子模块

| 函数 | 含义 |
|---|---|
| `COS(ctx, cfg)` | Couchbase **O**pen **S**tandard — 返回 `*Bucket` |
| `COC(ctx, cfg)` | Couchbase **O**pen **C**luster — 返回 `*Cluster` |
| `CPS(ctx, path)` | Couchbase **P**arse-and-Connect **S**tandard — YAML → `*Bucket` |
| `CPC(ctx, path)` | Couchbase **P**arse-and-Connect **C**luster — YAML → `*Cluster` |

**Bucket 方法**: `Get`、`Insert`、`Upsert`、`Remove`、`Ping`、`HealthCheck`

**追踪包装**: `GetTrace`、`InsertTrace`、`UpdateTrace`、`DeleteTrace` — 同名语义，自动生成 Span。

### Mongo 子模块

| 函数 | 含义 |
|---|---|
| `MOC(ctx, cfg)` | Mongo **O**pen **C**lient — 返回 `*Client` |
| `MOS(ctx, cfg)` | Mongo **O**pen **S**tandard — 返回 `*Collection` |
| `MPC(ctx, path)` | Mongo **P**arse-and-Connect **C**lient — YAML → `*Client` |
| `MPS(ctx, path)` | Mongo **P**arse-and-Connect **S**tandard — YAML → `*Collection` |

**Collection 方法**: `Find`、`FindOne`、`InsertOne`、`UpdateOne`、`DeleteOne`、`HealthCheck`

**追踪包装**: `FindTrace`、`FindOneTrace`、`InsertTrace`、`UpdateTrace`、`DeleteTrace`

---

## 示例

可运行的端到端示例位于 [`example/`](example/) 目录：

| 示例 | 说明 |
|---|---|
| [`01-jaeger`](example/01-jaeger) | Couchbase CRUD + Jaeger OTLP/gRPC 导出 |
| [`02-kafka-topic`](example/02-kafka-topic) | Couchbase CRUD + Kafka Topic 导出 |
| [`03-redis-stream`](example/03-redis-stream) | Couchbase CRUD + Redis Stream 导出 |
| [`04-noop`](example/04-noop) | Couchbase CRUD + `WithNoop()` 空追踪 |
| [`mongo_test`](example/mongo_test) | MongoDB CRUD + Jaeger OTLP/gRPC 导出 |

```bash
cd example/01-jaeger && go run .
```

---

## 构建与测试

所有命令均集成在顶层 [`Makefile`](Makefile) 中。

```bash
make build          # 构建根模块
make build-all      # 构建根模块 + 子模块
make test           # 运行单元测试（short 模式）
make test-all       # 运行所有模块的单元测试
make test-race      # 带 -race 检测器运行
make cover          # 测试覆盖率
make cover-html     # 交互式 HTML 覆盖率报告
make vet            # go vet ./...
make deps-check     # 检查分层依赖边界
make ci             # 完整 CI：deps → vet → openspec → test → coverage
```

---

## 依赖

| 库 | 用途 |
|---|---|
| `go.opentelemetry.io/otel` | OpenTelemetry 追踪 SDK |
| `confluentinc/confluent-kafka-go/v2` | Kafka 生产者（Kafka Topic 导出器） |
| `redis/go-redis/v9` | Redis 客户端（Redis Stream 导出器） |
| `couchbase/gocb/v2` | Couchbase Go SDK（仅 couchbase 子模块） |
| `go.mongodb.org/mongo-driver` | MongoDB Go 驱动（仅 mongo 子模块） |
| `gopkg.in/yaml.v3` | YAML 解析 |
| `google.golang.org/grpc` | OTLP/gRPC 传输层 |

---

## 迁移

如果从旧版本升级（旧版使用 `backend`/`mode`/`kafka_*` 扁平字段命名），请参见 [`MIGRATION.md`](MIGRATION.md) 获取完整的字段映射表。

---

## 贡献指南

1. 提交前运行 `make ci`——该命令会检查依赖边界、代码质量和测试覆盖率。
2. 保持分层架构：`observability/tracing` 不得导入 `mqx` 或任何子模块。
3. 遵循现有约定编写配置结构体、错误哨兵值和单例缓存。

---

## 许可

内部项目 — Space X HubX。
