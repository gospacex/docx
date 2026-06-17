# DbxWireJaeger

> 端到端参考示例:把 **dbx**(GORM v2)接入 **Wire** 注入器,并通过 OpenTelemetry Go SDK 把 span 导出到 **Jaeger**。[`../CachexWireJaeger`](../CachexWireJaeger) 的姊妹示例。

[![Status](https://img.shields.io/badge/status-stable--example-44cc11)]()
[![e2e](https://img.shields.io/badge/e2e-5%2F5%20PASS-44cc11)]()
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8)]()
[![OTel](https://img.shields.io/badge/OpenTelemetry-1.44-425cc7)]()
[![dbx](https://img.shields.io/badge/dbx-local%20replace-orange)]()
[![License](https://img.shields.io/badge/SPDX-LicenseRef--Internal--Use-lightgrey)]()

> [English version](./README.md) · 本文件是中文版,内容与英文版对齐。
> [查看英文版本](./README.md) · This file is the Chinese version, aligned with the English version.

---

## 目录

- [1. 这是一个什么示例](#1-这是一个什么示例)
- [2. 为什么需要这个示例](#2-为什么需要这个示例)
- [3. 快速开始](#3-快速开始)
- [4. 文件结构](#4-文件结构)
- [5. 架构](#5-架构)
  - [5.1 数据流](#51-数据流)
  - [5.2 调用方拥有 TracerProvider 的契约](#52-调用方拥有-tracerprovider-的契约)
  - [5.3 清理顺序](#53-清理顺序)
- [6. 配置参考(`mysql.yaml`)](#6-配置参考mysqlyaml)
- [7. Span operation 命名](#7-span-operation-命名)
- [8. 5 个用例](#8-5-个用例)
- [9. 与 `CachexWireJaeger` 的对比](#9-与-cachexwirejaeger-的对比)
- [10. Public API 参考](#10-public-api-参考)
  - [10.1 类型](#101-类型)
  - [10.2 构造函数](#102-构造函数)
  - [10.3 方法](#103-方法)
  - [10.4 错误语义](#104-错误语义)
- [11. 扩展](#11-扩展)
  - [11.1 加新用例](#111-加新用例)
  - [11.2 把 jaeger 换成别的 exporter](#112-把-jaeger-换成别的-exporter)
  - [11.3 跑非 legoB 的 MySQL](#113-跑非-legob-的-mysql)
- [12. 故障排查](#12-故障排查)
- [13. FAQ](#13-faq)
- [14. 边界(本例不做什么)](#14-边界本例不做什么)
- [15. 兼容矩阵](#15-兼容矩阵)
- [16. 版本固定](#16-版本固定)
- [17. 项目状态 / 稳定性](#17-项目状态--稳定性)
- [18. 贡献与变更日志](#18-贡献与变更日志)
- [19. 参考](#19-参考)
- [20. 许可](#20-许可)

---

## 1. 这是一个什么示例

一个独立的、单实例 MySQL 示例,跑通完整的 dbx 集成链路并接入真实的 Jaeger 采集器:

```
config.yaml ──► dbx.LoadConfig ──► dbsql.OpenPath ──► *gorm.DB
                                                            │
                                                            ▼
TestMain: OTel TracerProvider ──► otel.SetTracerProvider ──┘
                                       │
                                       ▼
                       GORM v2 callback (dbx/orm/gorm_tracing.go)
                                       │
                                       ▼
                            BatchSpanProcessor
                                       │
                                       ▼
                          OTLP HTTP → jaeger :4318
                                       │
                                       ▼
                            requireSpanReported(op)
                            GET jaeger :16686/api/traces
```

测试即契约:**5 个 GORM 操作 → 5 个根 span → 5 次 jaeger API 命中 → 5 个 PASS**。

## 2. 为什么需要这个示例

`hubx/examples/CachexWireJaeger/` 验证了 cachex + wire + jaeger 的链路。dbx 需要同样的验证,但接线在结构上不同:

| 关注点 | cachex (Redis) | **dbx (MySQL/GORM)** |
|---|---|---|
| 句柄 | `*redis.Client` | `*gorm.DB`(100% 原生) |
| Span 自动上报 | 需要 `redisotel.InstrumentTracing` | **自动** — 由 GORM v2 callback 触发(`dbx/orm/gorm_tracing.go`) |
| TracerProvider 初始化 | `cachex/initx.InitTracing`(一次调用) | **调用方拥有** — `dbsql.CreateExporter` + `sdktrace.NewTracerProvider` + `otel.SetTracerProvider` |
| Tracing 配置 | cachex 自有 `trace:` 块 | dbx 顶层 `tracing:` 块(`fileWrapper`);`dbsql.ExtractTracingAndApply` 是**文档化的 no-op** |
| 清理顺序 | `cleanup(ctx)` | `tpShutdown(ctx)` 在前,`sqlDB.Close()` 在后 — 反过来会丢 span |

这个示例的意义:任何要把 dbx 接入真实服务的人,直接复制这个目录就能拿到一份已知好用的模板。

## 3. 快速开始

```bash
# 一行命令。启动 legoB mysql + jaeger,等待就绪,跑 5 个 e2e 用例。
cd hubx/examples/DbxWireJaeger
make test
```

预期输出:

```
mysql ready (Health: healthy)
jaeger ready
ok  examples/dbxwirejaeger  2.983s
```

然后访问 <http://localhost:16686> 看 trace(service: `examples-dbxwirejaeger`)。

`make test` **不会** 在结束时 down 掉 legoB — 其他会话可以复用。只有在确实需要停的时候才用 `make down`。

## 4. 文件结构

```
DbxWireJaeger/
├── go.mod                module examples/dbxwirejaeger, Go 1.26
│                         replace dbx → 本地路径;pin 住 gorm v1.25.12
├── go.sum
├── wire.go               //go:build wireinject — injector 骨架
├── wire_gen.go           //go:build !wireinject — 由 `wire ./...` 生成
├── provider.go           DbxProvider.ProvideDB(cfgPath) (*gorm.DB, error)
├── model.go              User struct(gorm.Model + unique Name)+ newTestUser()
├── main_test.go          TestMain(OTel 三件套 + wire + AutoMigrate)
│                          + requireSpanReported 助手
│                          + 5 个 TestDbxWireJaeger_* 用例
├── mysql.yaml            mysql: + tracing: 两块(dbx fileWrapper schema)
├── Makefile              up / wait-mysql / wait-jaeger / run-tests / status / down
├── .gitignore            wire_gen.go.bak, *.test, *.out
├── README.md             (英文版,主入口)
└── README.zh.md          (本文件)
```

## 5. 架构

### 5.1 数据流

```
┌─────────────────────────────────────────────────────────────┐
│ TestMain (main_test.go)                                     │
│                                                             │
│  1. config.LoadMySQL(cfgPath)                               │
│       └─► (*MySQLConfig, *TracingConfig, error)             │
│                                                             │
│  2. dbsql.CreateExporter(ctx, traceCfg)                     │
│       └─► sdktrace.SpanExporter                             │
│                                                             │
│  3. sdkresource.New(ctx, WithAttributes(ServiceName(...)))  │
│       └─► *resource.Resource                                │
│                                                             │
│  4. sdktrace.NewTracerProvider(WithBatcher, WithResource)   │
│       └─► *TracerProvider                                   │
│       └─► otel.SetTracerProvider(tp)                        │
│       └─► tpShutdown := tp.Shutdown                         │
│                                                             │
│  5. InitializeInjector(cfgPath)                             │
│       └─► *Injector{DB: *DbxProvider}                       │
│                                                             │
│  6. injector.DB.ProvideDB(cfgPath)                          │
│       └─► *gorm.DB                                         │
│                                                             │
│  7. db.DB()  →  *sql.DB   (供清理时用)                      │
│                                                             │
│  8. sqlDB.PingContext(ctx)  (早期连通性检查)                 │
│                                                             │
│  9. db.AutoMigrate(&User{})  (一次性,所有用例共享)           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ TestDbxWireJaeger_Create  (与 4 个兄弟用例)                 │
│                                                             │
│  dbxDB.WithContext(ctx).Create(&u)   ──► GORM v2 callback   │
│                                          │                  │
│                                          ▼                  │
│                                dbx/orm/gorm_tracing.go      │
│                                tracer.Start(ctx, "db."+op)  │
│                                          │                  │
│                                          ▼                  │
│                                BatchSpanProcessor           │
│                                          │                  │
│                                          ▼                  │
│                                OTLP HTTP → jaeger :4318     │
│                                                             │
│  requireSpanReported(t, "db.create")                        │
│      GET :16686/api/traces?service=...&operation=db.create  │
│      200ms × 25 次重试,超时 fatal                            │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ 清理 (固定顺序,不可调换)                                    │
│                                                             │
│   tpShutdown(ctx)    ←  1. 先 flush span 缓冲               │
│   sqlDB.Close()      ←  2. 再关连接池                       │
│                                                             │
│  这两步反过来会丢 span。                                    │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 调用方拥有 TracerProvider 的契约

dbx 在设计上**不**为你安装 OTel TracerProvider。这是刻意的,且有文档说明(`dbsql/tracer.go:113-128`):`dbsql.ExtractTracingAndApply` 是 no-op。生产环境的 dbx 使用方必须自己接 trace。

`dbsql.CreateExporter` + `sdktrace.NewTracerProvider` + `otel.SetTracerProvider` 三件套是**唯一正确的接法**。别想偷懒 — `cachex/initx.InitTracing` 在这里不能复用(它解析的是 cachex 配置 schema,不是 dbx 的)。

### 5.3 清理顺序

```
tpShutdown(ctx)   →  把 BatchSpanProcessor 缓冲中的 span flush 到 OTLP exporter,
                      然后关闭 exporter
sqlDB.Close()     →  关闭 database/sql 连接池
```

如果先 `sqlDB.Close()`,`BatchSpanProcessor` 的最后一批 span 可能会尝试从一个已关闭的池里读 attribute,要么失败,要么用 `nil` resource 标 span。永远是 tp 在前,池在后。

## 6. 配置参考(`mysql.yaml`)

```yaml
mysql:                         # dbx MySQLConfig (dbx/config/config.go)
  host: localhost
  port: 3306
  username: testuser
  password: testpass           # legoB min profile 默认值;不要回显到日志
  database: testdb
  pool:
    max_open_conns: 20
    max_idle_conns: 5
    conn_max_lifetime: 1800    # 秒

tracing:                       # dbx TracingConfig (dbx/config/tracing.go)
  enabled: true
  service: examples-dbxwirejaeger    # 必须与 requireSpanReported 里的 serviceName 一致
  exporter: jaeger
  endpoint: localhost:4318          # OTLP HTTP, legoB min profile
  protocol: http
  sampler_type: always_on
  sampler_ratio: 1.0
```

字段引用(都从源码核对,不要靠猜):
- `dbx/config/config.go` — `MySQLConfig` 与 `BaseDBConfig`
- `dbx/config/tracing.go` — `TracingConfig`(注意:字段名是 `service`,**不是** cachex 的 `service_name`)
- `dbx/config/loader.go:14-25` — `fileWrapper`(`tracing:` 块在顶层,不在 `mysql:` 下面)

密码不会出现在任何错误路径里。`DbxProvider.ProvideDB` 用 `cfgPath` 包装,dbsql.OpenPath 自带的 native error;`password:` 字段永远不会进日志。

## 7. Span operation 命名

GORM v2 callback 在 `dbx/orm/gorm_tracing.go:84` 实际发出:

```go
ctx, span := tracer.Start(ctx, "db."+op, ...)
```

`op` 是 GORM callback 名。本示例断言的 4 个 operation:

| GORM 调用 | Span operation |
|---|---|
| `db.Create(&u)` | `db.create` |
| `db.First(&u, id)` | `db.query` |
| `db.Find(&[]User{})` | `db.query` |
| `db.Save(&u)` | `db.update` |
| `db.Delete(&User{}, id)` | `db.delete` |

> **坑点。** 从 cachex 样例复制过来的人会顺手写成 `gorm.*` — 错的。前缀是 `db.`,句号。如果 jaeger 搜索返回 0 条 trace,这是第一个要查的。

## 8. 5 个用例

每个用例对单个 GORM 操作跑一遍完整的 Arrange-Act-Assert-Verify 循环。

| 用例 | 操作 | 断言 span | 备注 |
|---|---|---|---|
| `TestDbxWireJaeger_Create` | `db.Create(&u)` | `db.create` | 校验自增 ID;`t.Cleanup` 删行 |
| `TestDbxWireJaeger_Get` | `db.First(&got, u.ID)` | `db.query` | 先种一行,再读回,断言 ID 一致 |
| `TestDbxWireJaeger_List` | `db.Find(&[]User{})` | `db.query` | 列表查询;不校验行数(其他用例的行可能存在) |
| `TestDbxWireJaeger_Update` | `db.Save(&u)` | `db.update` | 把 `Name` 追加 `-updated`;`t.Cleanup` 删行 |
| `TestDbxWireJaeger_Delete` | `db.Delete(&User{}, u.ID)` | `db.delete` | 用例自己的 `Delete` **就是** 清理,没有额外 `t.Cleanup` |

**隔离**:每个用例用 `newTestUser()`,`Name` 形如 `time.Now().UnixNano() + rand.Int63()`,保证并发/重复跑不撞 `uniqueIndex`。

**无 parent span**:每个用例都是 root span,与 cachex 例的 5 个独立 trace 平行。如果需要 parent(比如用 `user.crud` 做个 umbrella span 做 trace 分组),用 `go.opentelemetry.io/otel/trace` 的 `tracer.Start(ctx, "user.crud", ...)` 包一层 — 但那是偏离,不是扩展。

## 9. 与 `CachexWireJaeger` 的对比

| | CachexWireJaeger | **DbxWireJaeger** |
|---|---|---|
| 后端 | Redis(单实例) | MySQL(单实例) |
| 句柄 | `*redis.Client` | `*gorm.DB` |
| Trace 初始化 | `cachex/initx.InitTracing`(一次调用) | `dbsql.CreateExporter` + `sdktrace.NewTracerProvider` + `otel.SetTracerProvider`(三次调用) |
| Span 自动上报 | 需要 `redisotel.InstrumentTracing(cli)` | GORM v2 callback 自动触发 |
| 断言的 operations | `set / get / hset / hgetall / lpush / lrange / sadd / smembers / zadd / zrange` 等 | `db.create / db.query / db.update / db.delete` |
| 清理 | 一次 `cleanup(ctx)` | `tpShutdown` → `sqlDB.Close`,顺序固定 |
| 测试数据策略 | 短字符串 key(`k:hash`、`k:list`) | `User` 行 + unique `Name` 索引 |
| 适用场景 | 把 cachex 接入服务 | 把 dbx 接入服务 |

**怎么选**:
- Redis 缓存、pub/sub、有序集合、分布式锁 → `CachexWireJaeger`
- MySQL/PostgreSQL/SQLite 用 GORM + OTLP trace → `DbxWireJaeger`(本例)
- 同一个服务里两个都要 → 两个都拷,反正要两个 Provider

## 10. Public API 参考

本节枚举 `provider.go` 与 `wire.go` 导出的符号。**未在此列出的都是实现细节,仅供内部使用。**

### 10.1 类型

#### `type DbxProvider struct{}`

无状态 provider。并发安全(无字段,无需同步)。零值即可用;`NewDbxProvider` 是规范构造器,但 `&DbxProvider{}` 也行。

#### `type Injector struct{ DB *DbxProvider }`

Wire 输出结构体。`DB` 字段是导出的,以便 `wire.Struct(new(Injector), "*")` 能填它。`DB` 是唯一的注入目标 — 这个结构体的存在是为了给 wire 一个具体的返回类型。

### 10.2 构造函数

#### `func NewDbxProvider() *DbxProvider`

返回一个全新的、空的 provider。Wire 友好(无参)。

```go
dbxProvider := NewDbxProvider()
// 或者,放在 wire.Build set 里:
// wire.NewSet(NewDbxProvider)
```

#### `func InitializeInjector(cfgPath string) (*Injector, error)`

Wire 入口点。

**参数:**
- `cfgPath`(string):YAML 配置文件路径。**对 wire 的类型图来说,这个参数不使用** — 函数体里只有 `wire.Build(...)`。这个签名固定下来,是为了让 `TestMain` 在运行时能向 `ProvideDB` 传一致的值。

**返回:**
- `*Injector` — wire 后的 provider 图(在 `wire ./...` 生成的代码里)
- `error` — wire 内部错误;实际上对于本例这个简单图,基本是 `nil`

**Build tag:** `//go:build wireinject`。生成的对应物在 `wire_gen.go`,tag 是 `//go:build !wireinject`。

### 10.3 方法

#### `func (p *DbxProvider) ProvideDB(cfgPath string) (*gorm.DB, error)`

加载 YAML 配置,返回带连接池的原生 `*gorm.DB`。

**参数:**
- `cfgPath`(string,必填):YAML 文件的绝对或相对路径。必须能通过 dbx schema(mysql + tracing 两块)的校验。

**返回:**
- `*gorm.DB` — 已配置好、有连接池、可直接 `db.WithContext(ctx).Create(...)` 的句柄。GORM v2 callback 会把 span 发到全局 `TracerProvider`。
- `error` — 用 `cfgPath` 包装以方便调试。YAML 里的 `password:` 字段不会回显。

**副作用:**
- 读盘(`dbsql.OpenPath` → `config.Load` → `dbsql.Open`)
- 对配置的 MySQL 打开一个连接池
- **不**安装或修改任何 OTel TracerProvider(由调用方拥有,见 §5.2)

**线程安全:** 返回的 `*gorm.DB` 并发安全(参见 GORM v2 文档)。多个 goroutine 对同一个句柄发 `Create` / `Query` 等,共享底层的 `*sql.DB` 池。

**校验:**
- `cfgPath == ""` → 返回 `provide db: cfgPath is empty`
- `dbsql.OpenPath` 错误 → 用 cfgPath 上下文包装
- 其他 dbx 内部错误 → 原样传播

### 10.4 错误语义

`DbxProvider.ProvideDB` 用 `fmt.Errorf("provide db: %s: %w", ...)` 包装错误。`%w` 保留原始错误链,所以 `errors.Is` 和 `errors.As` 能穿过包装层。

`password` 字段刻意不出现在错误路径里。dbx 自己的错误可能含原始字段值;如果你需要脱敏,直接过滤 `mysql.yaml` 文件,而不是过滤 `DbxProvider` 输出。

## 11. 扩展

### 11.1 加新用例

1. 选一个 GORM 操作。先在源码里确认 span 名:`grep -n 'tracer.Start' /Users/hyx/work/gowork/src/lego2/dbx/orm/gorm_tracing.go`。
2. 写用例,按 4 段式(Arrange 种数据 → Act → Cleanup → Assert span)。
3. 用 `newTestUser()` 保证行唯一。
4. 末尾 `requireSpanReported(t, "db.<op>")`。
5. 跑 `make test`。新用例应当在对应 GORM 调用后 5 秒内通过。

### 11.2 把 jaeger 换成别的 exporter

改 `mysql.yaml` 的 `tracing:` 块:

```yaml
tracing:
  enabled: true
  service: examples-dbxwirejaeger
  exporter: kafka            # 或 redis_stream
  endpoint: localhost:9092   # 改成你 collector 的地址
  protocol: http
  ...
```

`dbsql.CreateExporter` 按 `exporter:` 字段路由,不需要改代码。`requireSpanReported` 助手硬编码了 jaeger — 换 exporter 后,把这个助手换成对你后端的断言。

### 11.3 跑非 legoB 的 MySQL

覆盖 `mysql.yaml`(别在原文件上改),把 `main_test.go` 里的 `cfgPath` 指到你的覆盖文件。legoB 是默认假设;换它是配置练习,不是代码练习。

换非 MySQL 后端(PostgreSQL、SQLite 等)的话,还要在 `go.mod` 里换 GORM driver,并调整 YAML 里的 `mysql:` 块字段 — schema 差异参见 dbx 文档。

## 12. 故障排查

| 症状 | 原因 | 修法 |
|---|---|---|
| `config.LoadMySQL "mysql.yaml": ...` | yaml 找不到或格式错 | 在示例目录下跑;确认 `mysql.yaml` 存在 |
| `dbsql.CreateExporter: ...` | tracing 配置非法 | 对照 `dbx/config/tracing.go` 检查 `tracing:` 块 |
| `mysql ping: ...` | legoB mysql 没起 | `docker compose -f /Users/hyx/work/gowork/src/legoB/docker-compose.yml up -d mysql` |
| `expected span operation="db.create" ...` | span 没到 jaeger | (1) 确认 `service:` 与 `main_test.go` 的 `serviceName` 一致。(2) 看 jaeger UI:<http://localhost:16686>。(3) 确认 `BatchSpanProcessor` 已配(TestMain 里已配)。 |
| `expected span operation="gorm.create" ...` | **你写错了前缀** | 改成 `db.create`。见第 7 节。 |
| `wire_gen.go` 与 `wire.go` 不一致 | injector 主体在 `wire.go` 改了 | 在示例目录下 `wire ./...`(必须 `GOWORK=off`) |
| `go mod tidy` 把 gorm 升到了 v1.25.x 之后 | 与 dbresolver v1.5.3 的 MVS 冲突 | 还原 `go.mod` 注释里写的 pin;别随便 `go mod tidy` |
| 5 个用例在 legoB 重启后都 `mysql ping: ...` 失败 | 连接池陈旧 | 再跑一次 `make test` — 下次 Ping 会重连 |
| `error: GOWORK=...` | 忘了 `GOWORK=off` 前缀 | Makefile 已经设了;如果直接 `go test`,手动加 `GOWORK=off` 前缀 |

## 13. FAQ

**Q: 为什么不写 `main.go`?**
A: 这是测试示例,不是可运行的二进制。`go test` 是唯一入口。如果你要可运行的二进制,把 `TestMain` 那段序列拷到自己的 `main.go` 里。

**Q: `InitializeInjector` 的 `cfgPath` 参数 wire 不用,为什么要保留?**
A: 保持签名一致 — `TestMain` 调 `InitializeInjector(cfgPath)` 然后调 `injector.DB.ProvideDB(cfgPath)`。参数是给运行时的,不是给 wire 类型图的。wire 规则要求函数体只能有 `wire.Build(...)`;`cfgPath` 出现在签名里但不在函数体里引用。

**Q: 能不能不改代码就加 kafka/redis_stream exporter?**
A: 可以 — 改 `mysql.yaml` 的 `tracing.exporter`。`dbsql.CreateExporter` 按这个字段路由。但 `requireSpanReported` 硬编码了 jaeger,所以你也要换掉断言助手。

**Q: 为什么用例 timeout 是 5 秒?**
A: jaeger 每 200ms 轮询一次,最多 25 次。5s 对健康的 `BatchSpanProcessor`(默认 5s schedule)够用。如果 SDK 默认调成 10s,这个会 flake — 简单修法:把 `jaegerWaitMax` 调大。

**Q: 为什么 gorm 要 pin 在 v1.25.12?**
A: `gorm.io/plugin/dbresolver v1.5.3` 引用了 `gorm.Stmt`,这个类型只在 gorm v1.25.x 里有。如果你放任 MVS 解析,gorm 会悄悄升到 v1.31.x,然后 dbx 依赖图编不过。保持 pin。

**Q: 能不能用 PostgreSQL 或 SQLite?**
A: dbx 都支持。要在本例里用,改 `go.mod` 里的 GORM driver,再调整 `mysql.yaml` 的字段。tracer 和 wire 代码不用动。5 个用例照常跑(GORM 在 API 层是 db 无关的)。

**Q: 为什么不加 parent span?**
A: 5 个独立的根 span 平行于 cachex 例的模式,而且让断言确定性更强。如果你要 parent(比如 `user.crud` umbrella),用 `go.opentelemetry.io/otel/trace` 的 `tracer.Start(ctx, "user.crud", ...)` 包一层。

**Q: `dbsql.OpenPath` 和 `dbsql.Open` 有什么区别?**
A: `OpenPath` 接文件路径;`Open` 接已经 parse 好的 `*MySQLConfig`。两者最终都到同一个 `*gorm.DB`。从 YAML 走时用 `OpenPath`;你已经有 config 的时候(比如 TestMain 里既需要 `*TracingConfig` 又需要 `*MySQLConfig`),用 `Open`。

**Q: 怎么只跑单个用例?**
A: `GOWORK=off go test -run TestDbxWireJaeger_Create -count=1 -race ./...` — 注意 TestMain 还是会跑完整初始化,所以这只省用例本体的时间,不省 setup 时间。

## 14. 边界(本例**不**做的事)

- **没有 `main.go`** — `go test` 是唯一入口。示例是 test 形态,不是二进制形态。
- **不用 dbresolver** — 只跑单实例。dbresolver 示例在 `dbx/examples/mysql-crud/`。
- **没有 kafka/redis_stream exporter** — 只有 jaeger。`tracing:` schema 支持别的;本例断言 jaeger。
- **没有 parent span / 事务用例** — 5 个根 span,与 cachex 的 5 个独立 trace 平行。
- **没有多库** — 只 MySQL。dbx 支持 8 个后端;本例只验证一个。
- **没有 `db.Take` / `db.Where` / `db.Updates` 变体** — 只覆盖 GORM CRUD 标准面。
- **无 README → 主 README** — 文档全在本文件;代码级使用提示在 `provider.go` 与 `wire.go` 的 `package main` doc comment 里。

## 15. 兼容矩阵

测过的版本:

| 组件 | 已测版本 | 最低版本 | 备注 |
|---|---|---|---|
| Go | 1.26.2 | 1.23 | examples 排除在 `lego2/go.work` 之外;各自有 go.mod |
| dbx | 本地 replace 自 `/Users/hyx/work/gowork/src/lego2/dbx` | n/a | replace 指令 pin 到本地路径 |
| gorm | 1.25.12 | 1.25.0 | **必须 1.25.x** —— dbresolver v1.5.3 要求 |
| dbresolver | 1.5.3 | 1.5.0 | 带来 gorm v1.25.x pin 要求 |
| OTel SDK | 1.44.0 | 1.40.0 | `semconv/v1.26.0` 要求 SDK ≥ 1.40 |
| semconv | v1.26.0 | v1.24.0 | 匹配 OTel SDK 1.44 的 resource attribute API |
| Wire | 0.7.0 | 0.6.0 | injector 主体规则在 0.6+ 不变 |
| MySQL | 8.4(legoB min profile) | 5.7 | GORM MySQL driver 支持 5.7+ |
| Jaeger | 1.63 OTLP HTTP | 1.35 | OTLP HTTP 接收器自 1.35 起可用 |
| Docker Compose | legoB | n/a | 外部基础设施,不属于本 module |

**矩阵外的情况:**
- GORM 1.26+ — 因为 dbresolver 引用了 `gorm.Stmt`(在 1.26+ 已删除),会编不过
- OTel SDK < 1.40 — `semconv/v1.26.0` 的 import path 可能不存在
- Wire 0.5.x 或更早 — injector 主体规则比这更早

## 16. 版本固定

```go
// go.mod (节选)
go 1.26.2
gorm.io/gorm v1.25.12  // 不要升级 — 见下
```

`gorm v1.25.12` 是故意 pin 住的。`gorm.io/plugin/dbresolver v1.5.3` 引用了 `gorm.Stmt`,这个类型只在 gorm v1.25.x 里有。如果你放任 MVS 解析,gorm 会悄悄升到 v1.31.x,然后 dbx 依赖图编不过。保持 pin。

OTel pin 在 `v1.44.0` 以匹配 `dbx/go.mod`。`semconv` 引入的是 `v1.26.0` — `semconv/v1.26.0` 是唯一与 OTel SDK 1.44 resource attribute API 匹配且稳定的一档。

Wire `v0.7.0` 是当前 stable;本例对 Wire 版本不敏感,0.6.0 起都行。

## 17. 项目状态 / 稳定性

**状态:稳定的示例,无活跃开发。**

这是参考示例,不是库。它"已经做完"了 —— 目的就是展示集成模式,不在活跃功能开发中。提的 issue 会按"是否影响文档准确性"分诊,不做功能请求。

**稳定性保证:**
- `examples/DbxWireJaeger/` 里的代码是示意性的;API 和模式随时可改
- `dbx`、`otelx`、`hubx` 库各自有 semver — 见各自 README
- legoB 是 fixture,通过该仓的 docker-compose 文件 pin 版本

**向后兼容:** 没有。这个目录的意义就是可拷可改,自由适配。

**弃用策略:** 没有。如果本例与未来的 dbx / hubx 模式脱节,会在新路径加一个后继者;本例不会事后改写。

## 18. 贡献与变更日志

### 变更日志

这是参考示例 —— 改动都记在 git 历史里:

```bash
git log -- examples/DbxWireJaeger/
```

没有单独的 `CHANGELOG.md`。重要的设计决策记录在 OpenSpec 归档里:

```
openspec/changes/archive/2026-06-17-examples-dbx-wire-jaeger/
```

### 贡献

1. 通过 sddflow 工作流(`/sddflow brainstorming`)提变更提案。
2. **不要**直接改这个示例 —— OpenSpec change 驱动实现。
3. 跑 `make test` 端到端验证你的改动。
4. 提评审前先跑 `openspec validate <change-name> --strict`。

纯文档改动(typo、README 措辞优化)可以直接改,在 commit message 里指明文件即可。

## 19. 参考

- **dbx 源码**(span 名 + tracing 契约的 ground truth):`/Users/hyx/work/gowork/src/lego2/dbx/`
  - `dbx/dbsql/tracer.go:113-128` — `ExtractTracingAndApply` 是 no-op
  - `dbx/orm/gorm_tracing.go:84` — `tracer.Start(ctx, "db."+op, ...)`
  - `dbx/config/loader.go:14-25` — `fileWrapper` schema
- **姊妹示例**:`hubx/examples/CachexWireJaeger/`
- **legoB min profile**:`/Users/hyx/work/gowork/src/legoB/docker-compose.yml`
- **OpenSpec change**(已归档):`openspec/changes/archive/2026-06-17-examples-dbx-wire-jaeger/`
- **外部资料**:
  - [GORM v2 文档](https://gorm.io/docs/) — GORM callback 模型
  - [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/) — TracerProvider、BatchSpanProcessor
  - [Jaeger OTLP receiver](https://www.jaegertracing.io/docs/1.63/apis/#opentelemetry-protocol-otlp) — endpoint 格式

## 20. 许可

**SPDX-License-Identifier: LicenseRef-Internal-Use**

本 module 是 lego2 monorepo 的一部分。**仅限内部使用** — 不对外发布、不再许可、不在 lego2 组织外再分发。

| 字段 | 值 |
|---|---|
| Copyright | (c) lego2 contributors |
| Maintainer | lego2 platform team |
| License | Internal use only(见上方 SPDX 标识) |
| 外部贡献 | 当前不接受 |

许可相关问题,走项目内部渠道联系 lego2 platform team。
