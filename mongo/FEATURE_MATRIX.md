# MongoDB 客户端功能矩阵

本文档仅描述 **当前实现并暴露** 给消费方的能力。driver 自身支持但未在
`mongo.Config` 或 `Collection` / `Client` 包装层暴露的特性不计入此表 —— 如果
你需要这些能力，请直接使用 [`go.mongodb.org/mongo-driver`](https://pkg.go.dev/go.mongodb.org/mongo-driver)
原语，或通过 `Client.Database(name).Collection(name)` 取得原始 collection 后调用。

## 当前暴露的能力

| 功能分类 | 功能点 | 配置项 | 说明 |
|---|---|---|---|
| **连接管理** | URI 连接 | `uri` | 通过 `options.Client().ApplyURI(cfg.URI)` 解析；支持单机、副本集、`mongodb+srv://` 等所有 driver URI 形式 |
| **连接管理** | 懒加载 + 单例 | — | 首次 `MOC` / `MOS` / `MPC` / `MPS` 调用时建立连接；后续调用按 fingerprint 复用 |
| **连接管理** | 连接池 | `max_pool_size` | 通过 `options.Client().SetMaxPoolSize` 设置；不设则用 driver 默认 |
| **连接管理** | 优雅关闭 | — | `Client.Close(ctx)` / `Collection.Close(ctx)` 释放单例条目并断开 driver 客户端 |
| **认证** | 用户名/密码 | `username`, `password` | 通过 `options.Credential` 设置，SCRAM-SHA-256 由 driver 默认协商 |
| **超时控制** | 连接超时 | `connect_timeout_ms` | 通过 `options.Client().SetConnectTimeout` 设置 |
| **健康检查** | Ping 健康检查 | — | `Client.HealthCheck(ctx)` / `Collection.HealthCheck(ctx)` |
| **生命周期** | 配置指纹 | — | `Config.CacheFingerprint()` / `Config.ContentHash()` 用于构造 cache key |
| **文档操作** | 读：`Find` / `FindOne` | — | `Collection.Find(ctx, filter)` 返回 `*mongo.Cursor`，`Collection.FindOne(ctx, filter)` 返回 `*mongo.SingleResult` |
| **文档操作** | 写：`InsertOne` / `UpdateOne` / `DeleteOne` | — | 返回 driver 原生 result 类型 |
| **文档操作** | 追踪包装 | — | `FindTrace` / `FindOneTrace` / `InsertTrace` / `UpdateTrace` / `DeleteTrace` 自动发出 OTel span |
| **追踪** | `observability.InitTracing` 集成 | `tracing` 块 | 与根模块的 `TracingConfig` 同构；详见根 `README.md` |

## 当前未在 config 层暴露（out-of-scope）

下列能力在底层 driver 中可用，但当前 SDK **没有** 在 `Config` 字段或
`Collection` 方法中包装。如需使用，请走 driver 原生 API：

- **复制集名称、副本集成员发现**：通过 URI 的 `replicaSet=...` 参数已隐式支持；SDK 不单独暴露
- **分片集群**：通过 URI 已隐式支持；SDK 不单独暴露
- **MongoDB SRV（DNS 发现）**：通过 `mongodb+srv://` URI 已支持；SDK 不单独暴露
- **TLS 配置（CA / 客户端证书 / 私钥）**：driver 通过 `options.Client().SetTLSConfig` 支持；SDK 未封装
- **x.509 / LDAP / AWS IAM / OIDC 认证**：driver 通过 `options.Credential.AuthMechanism` / `AuthSource` 支持；SDK 未封装
- **连接池最小大小、最大空闲时间**：driver 通过 `SetMinPoolSize` / `SetMaxConnIdleTime` 支持；SDK 未封装
- **Server 选择超时**：driver 通过 `SetServerSelectionTimeout` 支持；SDK 未封装
- **客户端操作超时（CSOT `timeoutMS`）**：driver 原生支持 context 取消；SDK 未封装单独的 MS 级超时
- **批量写入 BulkWrite**：driver 原生支持；SDK 未在 `Collection` 上包装
- **聚合管道 Aggregation**：driver 原生支持；SDK 未包装
- **索引管理**：driver 原生支持；SDK 未包装
- **Change Streams**：driver 原生支持；SDK 未包装
- **GridFS**：driver 原生支持；SDK 未包装
- **向量搜索（`bson.Vector`）**：driver 原生支持；SDK 未包装
- **Queryable Encryption / Search Index / IWM / zstd 压缩**：driver 原生支持；SDK 未包装
- **SDAM / 命令 / 连接池事件日志**：driver 原生支持；SDK 未封装

## 接口定义

### 入口点

| 函数 | 用途 | 返回 |
|---|---|---|
| `MOC(ctx, cfg)` | 从 `*Config` 获取 `*Client` | 单例 `*Client` 或 error |
| `MOS(ctx, cfg)` | 从 `*Config` 获取 `*Collection` | 单例 `*Collection` 或 error |
| `MPC(ctx, path)` | 从 YAML 路径获取 `*Client` | 单例 `*Client` 或 error |
| `MPS(ctx, path)` | 从 YAML 路径获取 `*Collection` | 单例 `*Collection` 或 error |

### `Client`

```go
type Client struct { /* unexported */ }
func (c *Client) Database(name string) *mongo.Database
func (c *Client) Collection(dbName, collName string) *Collection
func (c *Client) HealthCheck(ctx context.Context) error
func (c *Client) Close(ctx context.Context) error
func (c *Client) Config() *Config
```

### `Collection`

```go
type Collection struct { Name string }
func (c *Collection) Find(ctx, filter) (*mongo.Cursor, error)
func (c *Collection) FindOne(ctx, filter) *mongo.SingleResult
func (c *Collection) InsertOne(ctx, doc) (*mongo.InsertOneResult, error)
func (c *Collection) UpdateOne(ctx, filter, update) (*mongo.UpdateResult, error)
func (c *Collection) DeleteOne(ctx, filter) (*mongo.DeleteResult, error)
func (c *Collection) HealthCheck(ctx) error
func (c *Collection) Close(ctx) error
```

### Traced wrappers

```go
func FindTrace(ctx, coll, filter) (*mongo.Cursor, error)
func FindOneTrace(ctx, coll, filter) *mongo.SingleResult
func InsertTrace(ctx, coll, doc) (*mongo.InsertOneResult, error)
func UpdateTrace(ctx, coll, filter, update) (*mongo.UpdateResult, error)
func DeleteTrace(ctx, coll, filter) (*mongo.DeleteResult, error)
```

## 配置项汇总（仅暴露字段）

| 字段 | 类型 | 说明 |
|---|---|---|
| `uri` | string | MongoDB 连接 URI；driver 解析所有标准 URI 形式 |
| `database` | string | 默认数据库名（`MOS` / `MPS` 必填） |
| `collection` | string | 默认集合名（`MOS` / `MPS` 必填） |
| `username` | string | 用户名（与 `password` 同时设置时启用 SCRAM） |
| `password` | string | 密码 |
| `connect_timeout_ms` | int | 连接超时，毫秒 |
| `max_pool_size` | int | 连接池最大连接数 |
| `tracing` | block | `TracingConfig` 块，与根模块同构 |

## 依赖版本

| 库 | 版本 |
|---|---|
| `go.mongodb.org/mongo-driver` | v1.17.9 |
| `github.com/gospacex/hubx/cache/docx` | `0.2.0-dev` |
| Go | `1.26.2` |