# MongoDB 客户端功能矩阵

## MongoDB Go Driver 版本信息

- 当前支持版本：v1.x / v2.x（通过 `go.mongodb.org/mongo-driver`）
- 最低 Server 版本：MongoDB 4.4（v2.7+ 将需要 4.4+）
- Go 版本要求：Go 1.18+

## 功能矩阵

| 功能分类 | 功能点 | 实现状态 | 配置项 | 代码位置 |
|---------|--------|---------|--------|----------|
| **连接管理** | 单机连接 | ✅ 已实现 | `endpoints` | client.go:19-21 |
| **连接管理** | 副本集连接 | ✅ 已实现 | `replica_set` | client.go:49-51 |
| **连接管理** | 分片集群 | ⚠️ 待实现 | - | - |
| **连接管理** | MongoDB SRV (DNS) | ⚠️ 待实现 | `mongodb+srv://` | - |
| **连接管理** | 连接池 (Min/Max) | ✅ 已实现 | `pool_min_size`, `pool_max_size` | client.go:32-35 |
| **连接管理** | 连接池 MaxIdleTime | ✅ 已实现 | `pool_max_idle_time_ms` | - |
| **认证** | 用户名/密码认证 | ✅ 已实现 | `username`, `password` | client.go:69-72 (buildURI) |
| **认证** | SCRAM-SHA-256 | ✅ 已实现 | (默认) | - |
| **认证** | x.509 证书认证 | ⚠️ 待实现 | `tls_cert_file` | - |
| **认证** | LDAP 认证 | ⚠️ 待实现 | - | - |
| **认证** | AWS IAM 认证 | ⚠️ 待实现 | - | - |
| **认证** | OIDC 认证 (K8s) | ⚠️ 待实现 | - | - |
| **TLS** | TLS 基础支持 | ⚠️ 部分实现 | `tls.tls_enabled` | client.go:27-29 |
| **TLS** | TLS CA 证书 | ⚠️ 待实现 | `tls.tls_ca_file` | - |
| **TLS** | TLS 客户端证书 | ⚠️ 待实现 | `tls.tls_cert_file` | - |
| **TLS** | TLS 密钥文件 | ⚠️ 待实现 | `tls.tls_key_file` | - |
| **TLS** | InsecureSkipVerify | ⚠️ 待实现 | `tls.tls_insecure_skip_verify` | - |
| **超时控制** | 连接超时 | ✅ 已实现 | `connect_timeout_ms` | client.go:38-40 |
| **超时控制** | Socket 超时 | ✅ 已实现 | `socket_timeout_ms` | client.go:41-43 |
| **超时控制** | Server 选择超时 | ✅ 已实现 | `server_selection_timeout_ms` | client.go:44-46 |
| **超时控制** | 客户端操作超时 (CSOT) | ⚠️ 待实现 | `timeoutMS` | - |
| **健康检查** | Ping 健康检查 | ✅ 已实现 | - | client.go:74-76 |
| **生命周期** | 懒加载初始化 | ✅ 已实现 | - | client.go:54 |
| **生命周期** | 优雅关闭 | ✅ 已实现 | - | client.go:78-83 |
| **生命周期** | 批量关闭所有实例 | ✅ 已实现 | - | instance.go:54-66 |
| **数据库操作** | 获取 Database | ✅ 已实现 | `database` | client.go:85-87 |
| **数据库操作** | 获取 Collection | ✅ 已实现 | - | client.go:89-91 |
| **数据库操作** | CRUD 操作 | ⚠️ 待实现 | - | (返回原生 client) |
| **数据库操作** | 批量写操作 (BulkWrite) | ⚠️ 待实现 | - | - |
| **数据库操作** | 聚合操作 | ⚠️ 待实现 | - | - |
| **数据库操作** | 索引管理 | ⚠️ 待实现 | - | - |
| **数据库操作** | Change Streams | ⚠️ 待实现 | - | - |
| **数据库操作** | GridFS | ⚠️ 待实现 | - | - |
| **特性** | 向量搜索 (Vector Search) | ⚠️ 待实现 | `bson.Vector` | - |
| **特性** |Queryable Encryption | ⚠️ 待实现 | - | - |
| **特性** | 搜索索引管理 | ⚠️ 待实现 | - | - |
| **特性** | 智能负载管理 (IWM) | ⚠️ 待实现 | - | - |
| **特性** | 压缩 (zstd) | ⚠️ 待实现 | - | - |
| **日志** | SDAM 日志 | ⚠️ 待实现 | - | - |
| **日志** | 命令日志 | ⚠️ 待实现 | - | - |
| **日志** | 连接池事件 | ⚠️ 待实现 | - | - |

## 配置项汇总

| 配置项 | 类型 | 说明 | 状态 |
|--------|------|------|------|
| `endpoints` | []string | MongoDB 连接地址 | ✅ |
| `database` | string | 默认数据库名 | ✅ |
| `username` | string | 认证用户名 | ✅ |
| `password` | string | 认证密码 | ✅ |
| `auth_source` | string | 认证来源数据库 | ✅ |
| `replica_set` | string | 副本集名称 | ✅ |
| `pool_min_size` | int | 连接池最小连接数 | ✅ |
| `pool_max_size` | int | 连接池最大连接数 | ✅ |
| `pool_max_idle_time_ms` | int | 连接最大空闲时间(ms) | ✅ |
| `connect_timeout_ms` | int | 连接超时(ms) | ✅ |
| `socket_timeout_ms` | int | Socket 超时(ms) | ✅ |
| `server_selection_timeout_ms` | int | Server 选择超时(ms) | ✅ |
| `tls.tls_enabled` | bool | 是否启用 TLS | ⚠️ |
| `tls.tls_ca_file` | string | CA 证书路径 | ❌ |
| `tls.tls_cert_file` | string | 客户端证书路径 | ❌ |
| `tls.tls_key_file` | string | 私钥路径 | ❌ |
| `tls.tls_insecure_skip_verify` | bool | 跳过证书验证 | ❌ |

## 接口定义

### Instance 接口
```go
type Instance interface {
    HealthCheck(ctx context.Context) error
    Close() error
}
```

### Provider 接口
```go
type Provider interface {
    Name() string
    Type() TypeID
    Build(instanceName string, cfg map[string]any) (Instance, error)
    HealthCheck(ctx context.Context) error
    Close() error
}
```

## 状态说明

| 状态 | 说明 |
|------|------|
| ✅ 已实现 | 功能完整可用 |
| ⚠️ 待实现 | 框架已预留，需扩展实现 |
| ❌ 未实现 | 完全未支持 |