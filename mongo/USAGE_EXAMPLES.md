# MongoDB 客户端使用案例

所有示例都基于 `github.com/gospacex/hubx/cache/mongo` **当前实现** 的入口点：
`MOC` / `MOS` / `MPC` / `MPS`，以及 `Collection` 上的 CRUD 包装。

完整 API 签名见 [`FEATURE_MATRIX.md`](FEATURE_MATRIX.md)。

---

## 1. 基础连接

### 1.1 通过 Go 结构体连接

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/gospacex/hubx/cache/docx/config"
    "github.com/gospacex/hubx/cache/docx/observability"
    mongo "github.com/gospacex/hubx/cache/mongo"
)

func main() {
    ctx := context.Background()

    // 显式初始化追踪；后续 traced 包装会自动发出 span。
    if err := observability.InitTracing(ctx, config.TracingConfig{
        Enabled:     true,
        ServiceName: "mongo-demo",
        Exporter:    "jaeger",
        Endpoint:    "localhost:4317",
        Protocol:    "grpc",
    }); err != nil {
        log.Fatal(err)
    }
    defer observability.ShutdownTracing(ctx)

    cfg := &mongo.Config{
        URI:            "mongodb://localhost:27017",
        Database:       "myapp",
        Collection:     "users",
        ConnectTimeout: 5000,  // ms
        MaxPoolSize:    50,
        // Username / Password 留空表示无认证
    }

    coll, err := mongo.MOS(ctx, cfg)  // MOS 返回 *Collection（单例）
    if err != nil {
        log.Fatalf("MOS: %v", err)
    }
    defer coll.Close(ctx)

    fmt.Printf("connected to collection=%s\n", coll.Name)
}
```

### 1.2 副本集 / SRV / 其他 URI 形式

URI 解析完全交给 `mongo-driver`，SDK 只在 `Config.URI` 上原样转发。
`replicaSet=...`、`mongodb+srv://...`、`readPreference=...` 等都通过 URI 表达：

```go
cfg := &mongo.Config{
    URI:        "mongodb+srv://user:pass@cluster0.example/admin?retryWrites=true",
    Database:   "myapp",
    Collection: "orders",
}
coll, err := mongo.MOS(ctx, cfg)
```

> 注意：副本集成员发现、SRV DNS、读偏好等全部由 driver 处理；当前 SDK
> **不在 `Config` 字段层暴露** 这些选项（详见 `FEATURE_MATRIX.md` 的
> "out-of-scope" 列表）。

---

## 2. 认证

### 2.1 SCRAM 用户名/密码

```go
cfg := &mongo.Config{
    URI:        "mongodb://node1:27017,node2:27017",
    Database:   "myapp",
    Collection: "users",
    Username:   "admin",
    Password:   "${env:MONGO_PASSWORD}",   // ExpandEnvVars 由 ParseConfig 处理
}
coll, err := mongo.MOS(ctx, cfg)
```

如果 `Username` / `Password` 同时非空，SDK 会调用 `options.Credential`，
driver 默认走 SCRAM-SHA-256。

### 2.2 YAML 形式

```yaml
# mongo.yaml
uri: mongodb://localhost:27017
database: myapp
collection: users
username: admin
password: ${env:MONGO_PASSWORD}
connect_timeout_ms: 5000
max_pool_size: 50
```

```go
coll, err := mongo.MPS(ctx, "mongo.yaml")  // MPS = Mongo Parse-and-Standard
```

> x.509 / LDAP / AWS IAM / OIDC 等认证机制当前未在 SDK 层封装；如需使用请
> 走 `options.Credential.AuthMechanism` 原生 API。

---

## 3. 从 YAML 直接获得 `*Client`

如果需要多个 database / collection，建议拿 `*Client` 再自行打开 collection：

```go
client, err := mongo.MPC(ctx, "mongo.yaml")
if err != nil { log.Fatal(err) }
defer client.Close(ctx)

users := client.Collection("myapp", "users")
orders := client.Collection("myapp", "orders")

_ = users
_ = orders
```

`MPC` / `MPS` 都会用 `Config.CacheFingerprint()` 构造缓存 key；多个调用
只要 config 相同就共享同一连接。

---

## 4. CRUD（无追踪）

`Collection` 上暴露的最小 CRUD 直接转发到 driver：

```go
coll, _ := mongo.MOS(ctx, cfg)
ctx := context.Background()

// InsertOne
res, err := coll.InsertOne(ctx, map[string]any{"name": "张三", "age": 30})

// FindOne
single := coll.FindOne(ctx, map[string]any{"name": "张三"})
var got map[string]any
if err := single.Decode(&got); err != nil { /* ... */ }

// Find
cursor, err := coll.Find(ctx, map[string]any{"age": map[string]any{"$gte": 18}})
if err != nil { /* ... */ }
defer cursor.Close(ctx)
for cursor.Next(ctx) { /* ... */ }

// UpdateOne / DeleteOne
_, err = coll.UpdateOne(ctx,
    map[string]any{"name": "张三"},
    map[string]any{"$set": map[string]any{"age": 31}})
_, err = coll.DeleteOne(ctx, map[string]any{"name": "张三"})
```

---

## 5. CRUD（带追踪）

```go
import "github.com/gospacex/hubx/cache/mongo"

cursor, err := mongo.FindTrace(ctx, coll, map[string]any{"name": "张三"})
if err != nil { /* ... */ }
defer cursor.Close(ctx)

_, err = mongo.InsertTrace(ctx, coll, map[string]any{"name": "李四"})
_, err = mongo.UpdateTrace(ctx, coll,
    map[string]any{"name": "李四"},
    map[string]any{"$set": map[string]any{"age": 28}})
_, err = mongo.DeleteTrace(ctx, coll, map[string]any{"name": "李四"})
```

`FindTrace` / `InsertTrace` / `UpdateTrace` / `DeleteTrace` 在调用前会先开
span（span name 见下表），调用结束后自动 `End()`，driver 报错时通过
`span.RecordError(err)` 标注。

| Traced 函数 | Span name |
|---|---|
| `FindTrace` | `mongo.Find` |
| `FindOneTrace` | `mongo.FindOne` |
| `InsertTrace` | `mongo.InsertOne` |
| `UpdateTrace` | `mongo.UpdateOne` |
| `DeleteTrace` | `mongo.DeleteOne` |

每个 span 都带 `collection=<name>` 属性，便于在 trace UI 里按集合过滤。

---

## 6. 健康检查

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

if err := coll.HealthCheck(ctx); err != nil {
    log.Printf("health check failed: %v", err)
}
```

`Collection.HealthCheck` 内部委托给 `client.Ping(ctx, nil)`；`Client` 也
暴露同名方法，等价。

---

## 7. 优雅关闭

```go
defer coll.Close(ctx)   // 单例条目 evict + 底层 client.Disconnect
// 或
defer client.Close(ctx)
```

`Close` 会调用 driver 的 `Disconnect`，并从单例缓存里删除当前 key。注意：
如果 `MOC` 之后又调用了 `MOS`，两者共享同一个底层 `*Client`；`Close` 任意
一次都会断开该 client，其他持有引用者后续调用会失败。

---

## 8. 常见错误处理

`MOC` / `MOS` / `MPC` / `MPS` 的错误都带包名前缀 `mongo:`，可按字符串匹配
做分支（也鼓励用 `errors.Is` 在你自定义错误上）：

```go
coll, err := mongo.MOS(ctx, cfg)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "URI is required"):
        log.Fatal("配置错误：必须指定 uri")
    case strings.Contains(err.Error(), "database is required"):
        log.Fatal("配置错误：MOS/MPS 模式必须指定 database + collection")
    case strings.Contains(err.Error(), "connect"):
        log.Printf("连接错误：%v", err)
    default:
        log.Fatal(err)
    }
}
```

更推荐的方式是把 `Config.CacheFingerprint()` 在调用前先跑一遍 —— 任何
YAML 字段缺失都会在这里失败，错误信息精确到字段名。

---

## 9. 与其他客户端对比

> 仅对比当前 SDK 实际暴露的能力。

| 维度 | `cache/redis`（如未来提供） | `cache/mongo` |
|---|---|---|
| 连接池 | ✅ | ✅（`max_pool_size`） |
| 认证 | ✅ | ✅（SCRAM 用户名/密码） |
| TLS | ✅ | ❌（需走 driver 原生） |
| 单机 / 副本集 | ✅ | ✅（URI 表达） |
| 健康检查 | ✅ | ✅ |
| 显式 TracerProvider | ✅ | ✅ |