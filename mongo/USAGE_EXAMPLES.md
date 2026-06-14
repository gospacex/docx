# MongoDB 客户端使用案例

## 1. 基础连接

### 1.1 单机连接

```yaml
hubx:
  clients:
    mongo:
      default:
        endpoints: ["mongodb://localhost:27017"]
        database: "myapp"
        connect_timeout_ms: 5000
```

```go
// 获取默认实例
inst, err := mongo.GetClient("mongo", "default")
if err != nil {
    log.Fatal(err)
}
defer inst.Close()

// 直接使用原生 mongo.Client
coll := inst.Collection("myapp", "users")
ctx := context.Background()
err = coll.InsertOne(ctx, bson.M{"name": "张三", "age": 30})
```

### 1.2 副本集连接

```yaml
hubx:
  clients:
    mongo:
      replica-set:
        endpoints:
          - "mongodb://node1:27017"
          - "mongodb://node2:27017"
          - "mongodb://node3:27017"
        replica_set: "rs0"
        database: "myapp"
        pool_max_size: 50
```

```go
inst, _ := mongo.GetClient("mongo", "replica-set")
coll := inst.Collection("myapp", "orders")
```

## 2. 认证配置

### 2.1 用户名密码认证

```yaml
hubx:
  clients:
    mongo:
      authenticated:
        endpoints: ["mongodb://localhost:27017"]
        username: "admin"
        password: "secret"
        auth_source: "admin"
        database: "myapp"
```

### 2.2 TLS 连接

```yaml
hubx:
  clients:
    mongo:
      secure:
        endpoints: ["mongodb://localhost:27017"]
        database: "myapp"
        tls:
          tls_enabled: true
          tls_ca_file: "/path/to/ca.crt"
          tls_cert_file: "/path/to/client.crt"
          tls_key_file: "/path/to/client.key"
        connect_timeout_ms: 10000
```

## 3. 连接池配置

### 3.1 高并发场景

```yaml
hubx:
  clients:
    mongo:
      high-concurrency:
        endpoints: ["mongodb://localhost:27017"]
        database: "myapp"
        pool_min_size: 10
        pool_max_size: 100
        pool_max_idle_time_ms: 30000
        connect_timeout_ms: 5000
        socket_timeout_ms: 30000
```

### 3.2 低流量场景

```yaml
hubx:
  clients:
    mongo:
      low-traffic:
        endpoints: ["mongodb://localhost:27017"]
        database: "myapp"
        pool_min_size: 1
        pool_max_size: 5
        pool_max_idle_time_ms: 60000
```

## 4. 超时配置

```yaml
hubx:
  clients:
    mongo:
      timeout-config:
        endpoints: ["mongodb://localhost:27017"]
        database: "myapp"
        connect_timeout_ms: 5000           # 5s 连接超时
        socket_timeout_ms: 30000           # 30s Socket 超时
        server_selection_timeout_ms: 3000 # 3s Server 选择超时
```

## 5. 健康检查

```go
// 使用 Instance 接口进行健康检查
inst, _ := mongo.GetClient("mongo", "default")

ctx := context.Background()
err := inst.HealthCheck(ctx)
if err != nil {
    log.Printf("MongoDB health check failed: %v", err)
}
```

## 6. 批量关闭

```go
// 关闭所有 MongoDB 客户端实例
func shutdownMongo() {
    if err := mongo.CloseAll(); err != nil {
        log.Printf("Error closing mongo clients: %v", err)
    }
}
```

## 7. Provider 注册

```go
import "gospacex/hubx/db/mongo"

// 在初始化时注册 Provider
func init() {
    hubx.RegisterClient(mongo.NewProvider())
}

// 通过 hubx 获取实例
inst, err := hubx.GetClient("mongo", "default")
```

## 8. 配置代码示例

### 8.1 通过 map 配置

```go
cfg := map[string]any{
    "endpoints": []string{"mongodb://localhost:27017"},
    "database":  "myapp",
    "username":  "admin",
    "password":  "secret",
    "auth_source": "admin",
    "pool_max_size": 50,
    "connect_timeout_ms": 5000,
}

inst, err := mongo.NewProvider().Build("default", cfg)
if err != nil {
    log.Fatal(err)
}
```

### 8.2 使用 UnmarshalConfig

```go
var cfg mongo.Config
err := mongo.UnmarshalConfig(cfgMap, &cfg)
if err != nil {
    log.Fatal(err)
}
```

## 9. 常见错误处理

```go
inst, err := mongo.GetClient("mongo", "default")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "endpoints is required"):
        log.Fatal("配置错误: 必须指定 endpoints")
    case strings.Contains(err.Error(), "connect"):
        log.Fatal("连接错误: 无法连接到 MongoDB")
    default:
        log.Fatal(err)
    }
}
```

## 10. 与其他客户端对比

### 10.1 Redis vs MongoDB

| 维度 | Redis | MongoDB |
|------|-------|---------|
| 用途 | KV 缓存 | 文档数据库 |
| 连接池 | ✅ | ✅ |
| 认证 | ✅ | ✅ |
| TLS | ✅ | ⚠️ 部分 |
| 健康检查 | ✅ | ✅ |
| 单机/集群 | ✅ | ✅ (副本集) |

### 10.2 GORM vs MongoDB

| 维度 | GORM (MySQL/Postgres) | MongoDB |
|------|----------------------|---------|
| 数据模型 | 关系型 | 文档型 |
| 连接池 | ✅ | ✅ |
| 事务 | ✅ | ⚠️ 待实现 |
| 关联查询 | ✅ | ⚠️ 待实现 |
| 聚合 | ❌ | ⚠️ 待实现 |