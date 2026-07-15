# Development: 架构规范与性能规范

## 1. 禁止 N+1 查询 (No N+1 Query)

所有数据库访问必须提前设计查询策略，这是**强制约束**。

### 1.1 禁止模式

- 在循环、`forEach`、`map`、`range` 内执行数据库查询。
- 在列表接口中逐条查询关联数据。
- 查询列表 → 循环查询详情
- 查询用户列表 → 循环查询用户角色
- 查询设备列表 → 循环查询设备状态
- 查询任务列表 → 循环查询算法包信息

**错误示例**：
```go
devices, _ := repo.List(ctx)
for _, device := range devices {
    status, _ := repo.GetStatus(ctx, device.ID)
    device.Status = status
}
```

### 1.2 正确方式

通过单次查询获取完整数据：

```go
devices, _ := repo.ListWithStatus(ctx)
```

使用策略（任选或组合）：

- SQL `JOIN`
- GORM `Preload`
- 批量查询 `WHERE id IN (...)`
- 聚合查询
- 数据缓存

## 2. Repository 查询范围规范

- 每个 Repository 方法必须有明确的查询范围。
- 禁止：
  - 无条件查询全表
  - 隐式加载关联对象
  - Service 层拼接数据库查询

## 3. 列表 API 规范

所有列表接口必须内置支持：

- **分页**（page + pageSize，含总数）
- **排序**（可排序字段与排序方向）
- **条件过滤**（按业务维度筛选）
- **关联数据预加载策略**（明确 Preload/Join 范围）

## 4. 新增数据库查询代码核查项

新增数据访问代码时必须评估以下风险：

- [ ] SQL 执行次数（单次 vs 多次）
- [ ] 是否存在循环查询风险
- [ ] 是否使用了有效的索引
- [ ] 是否可能在大数据量下造成性能问题

## 5. 禁止隐藏查询

- 禁止 GORM Lazy Loading 思维（隐式触发 SQL）
- 禁止 Model 方法内部访问 DB
- 禁止 JSON 序列化触发额外查询

## 6. Go Backend 数据库操作规范

数据访问必须严格遵循分层单向依赖：

```
Handler → Service → Repository → GORM Model
```

禁止跨层访问 DB。

### 6.1 Repository 职责

Repository 层负责：

- 查询组合与条件构造
- JOIN 关联查询
- Preload 预加载
- 聚合查询（Group、Count、Sum）
- 分页与排序
- 条件过滤

### 6.2 Service 禁止

- 调用多个 Repository 手动拼装关联数据造成 N+1
- 循环调用 Repository
- 根据业务对象再次查询数据库

### 6.3 GORM 使用示例

**允许（Preload 关联）**：
```go
db.Preload("Algorithms").
   Preload("Streams").
   Find(&devices)
```

**允许（批量查询）**：
```go
db.Where("id IN ?", ids).Find(&items)
```

**允许（聚合查询）**：
```go
db.Model(&Device{}).
   Select("status, count(*)").
   Group("status").
   Scan(&stats)
```

**禁止（循环查询）**：
```go
for _, item := range items {
    db.First(&detail, item.ID)
}
```

**禁止（Model 持有数据库逻辑）**：
```go
func (u User) Roles() []Role {
    db.Where(...).Find(...)
}
```

## 7. Pull Request Review Checklist

提交代码前必须逐项确认。

### Database

- [ ] 是否存在循环中的数据库访问？
- [ ] 是否可能产生 N+1 查询？
- [ ] 列表接口是否一次性加载关联数据？
- [ ] 是否使用分页？
- [ ] 是否有必要的数据库索引？
- [ ] 是否避免 `SELECT *`？
- [ ] 是否避免大表全量扫描？

### Performance

- [ ] 是否增加数据库 Round Trip？
- [ ] 是否可以批量处理？
- [ ] 是否需要 Redis 缓存？
- [ ] 是否影响边缘设备资源？

## 8. 缓存策略

项目采用 Redis 分布式缓存 + MemoryCache 本地缓存双层架构。所有缓存操作必须遵循以下规范。

### 8.1 缓存接口使用规范

使用 `app/internal/pkg/cache.Cache` 接口（不要直接用 `*redis.Client` 做 KV 缓存）。

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Del(ctx context.Context, key string) error
}
```

- 值统一 JSON 序列化为 `[]byte`，调用方手动 `json.Marshal`/`json.Unmarshal`。
- 缓存未命中返回 `cache.ErrCacheMiss`，不要用空值或零值区分。
- RedisCache 与 MemoryCache 共用同一接口，切换对业务透明。

### 8.2 缓存读策略：Cache-Aside（首选）

所有读操作用 Cache-Aside 模式：

```go
// 1. 读缓存
cached, err := s.cache.Get(ctx, key)
if err == nil {
    // 2. 命中 → 反序列化返回
    json.Unmarshal(cached, &result)
    return result, nil
}

// 3. 未命中 → 查数据库
result, err := s.repo.Find(ctx, ...)

// 4. 写缓存（带 TTL）
data, _ := json.Marshal(result)
s.cache.Set(ctx, key, data, ttl)
return result, nil
```

### 8.3 缓存一致性：写入时失效

数据变更时**先写数据库，再删除缓存**（Cache-Aside + 显式失效）。

**正确**：
```go
// 事务内更新数据库
db.Save(&record)
// 提交后删除缓存
s.cache.Del(ctx, cacheKey)
```

**禁止**：
- 先删缓存再写 DB（并发读到旧数据）
- 更新缓存值而非删除（无法保证序列化一致性）
- 只写缓存不写 DB（缓存即真相源只在明确设计的场景使用）

**批量失效**：当多个 Key 共享前缀时，使用 `SCAN` 批量删除：
```go
iter := s.rdb.Scan(ctx, 0, "perm:*", 0).Iterator()
for iter.Next(ctx) {
    s.rdb.Del(ctx, iter.Val())
}
```

### 8.4 缓存 Key 命名规范

格式：`namespace:subtype[:specific_id][:qualifier]`

- 冒号 `:` 分隔，全部小写，无空格
- 先写命名空间，再写资源类型，最后写业务标识
- 长/派生 Key 使用 SHA256 截断前 8 字节作为后缀

**已有约定**：

| 用途 | Key 示例 | TTL |
|------|---------|-----|
| 权限树 | `perm:tree:all` | 10 min |
| 菜单树 | `perm:menu_tree:<hash8>` | 10 min |
| 权限码 | `perm:codes:<hash8>` | 10 min |
| 用户权限 | `perm:u_<userID>` | 5 min |
| 设备状态 | `device:status:<id>` | 24 h |
| 心跳 ZSet | `aivision:edge:heartbeats` | (score 过期) |
| 防抖窗口 | `debounce:alarm:<dev>:<algo>:<type>` | 5 s |
| 限流 | `rate_limit:<clientIP>` | 1 min |
| 刷新令牌 | `refresh:<userID>:<tokenUUID>` | 跟随 Token TTL |

新增缓存 Key 时必须：
- 在代码中用常量而非字符串字面量
- 保持命名空间前缀一致（如权限相关用 `perm:`）
- 在 `app/internal/pkg/cache/keys.go` 或对应 service 文件顶部集中定义

### 8.5 TTL 策略

| 数据类型 | TTL 建议 | 说明 |
|---------|---------|------|
| 静态配置/元数据 | 10-30 min | 权限树、菜单树、角色列表 |
| 用户关联数据 | 3-5 min | 用户权限列表、RBAC 缓存 |
| 实时状态 | 15-60 s | 设备在线状态、心跳标记 |
| 防抖/限流 | 秒级（5-60 s) | 防抖窗口、Rate Limit |
| 分布式锁 | 秒级（<60 s) | 45s 以内，避免锁残留 |
| 设备持久状态 | 24 h | 设备静态属性（类型、固件版本） |
| 空值占位 | 30-60 s | 防止缓存穿透 |

- 所有缓存必须设置 TTL，禁止无过期时间的 Key（分布式锁除外）。
- 读多写少的数据使用长 TTL + 写入时显式失效策略。
- 高频变化的数据使用短 TTL（15-60 s），避免强一致性要求。

### 8.6 缓存穿透防护

对于数据库中不存在的数据：

```go
// 缓存空值占位，短 TTL 防止穿透
if errors.Is(err, gorm.ErrRecordNotFound) {
    s.cache.Set(ctx, key, nilBytes, 30*time.Second)
    return nil, ErrNotFound
}
```

适用场景：高频查询的热点 Key 在 DB 中无对应记录（如无效的 deviceID 重复查询）。

### 8.7 缓存雪崩/击穿防护

- **缓存雪崩**（大量 Key 同时过期）：TTL 添加随机偏移，避免同一时刻集中失效。
  ```go
  ttl := baseTTL + time.Duration(rand.Intn(60))*time.Second
  ```
- **缓存击穿**（热点 Key 过期后高并发回源）：使用 Redis `SETNX` 互斥锁控制回源：
  ```go
  // 只有一个 goroutine 查 DB，其余等待
  lockKey := "lock:" + cacheKey
  acquired, _ := s.rdb.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
  if acquired {
      defer s.rdb.Del(ctx, lockKey)
      data, _ := loadFromDB(ctx)
      s.cache.Set(ctx, cacheKey, data, ttl)
  } else {
      time.Sleep(50 * time.Millisecond)
      cached, err := s.cache.Get(ctx, cacheKey)
  }
  ```
- **高扇出查询**（如 N 个设备同时查各自的缓存）：确保批量接口（`BatchIsOnline`/`WHERE IN`）优先于逐条查询 + 缓存。

### 8.8 本地缓存 (MemoryCache) 使用规范

`app/internal/pkg/cache.MemoryCache` 适用于：

- 开发/测试环境无 Redis 时的兜底
- 几乎不变且不跨实例共享的元数据（如算法包版本列表）
- 性能敏感且允许分钟级不一致的数据

**必须**：
- 使用 `NewMemoryCacheWithContext(ctx, interval)` 管理生命周期，避免 goroutine 泄漏
- 或显式调用 `Stop()` 清理 janitor goroutine
- 明确注释声明一致性容忍度

**禁止**：
- 用本地缓存替代 Redis 做分布式一致性场景
- 在需要跨节点实时同步的数据上使用 MemoryCache

### 8.9 直接 Redis 操作规范

以下场景允许绕过 `Cache` 接口直接使用 `*redis.Client`：

| 场景 | 命令 | 说明 |
|------|------|------|
| 分布式锁 | `SETNX` + Lua DEL | 互斥控制 |
| 有序集合 | `ZADD`/`ZRANGE`/`ZSCORE` | 心跳跟踪、排行榜 |
| 计数器 | `INCR` + `EXPIRE` | 限流、统计 |
| 发布订阅 | `PUBLISH`/`SUBSCRIBE` | 跨实例通知 |
| 原子获取并删除 | `GETDEL` | 一次性令牌消费 |

所有直接操作必须：
- 使用 Lua 脚本保证比较-删除等复合操作的原子性
- 设置合理的超时（`DialTimeout: 5s`, `ReadTimeout: 3s`, `WriteTimeout: 3s`）
- 处理 Redis 连接失败时的降级策略（fail-open 或 fail-close，视场景决定并注释说明）

**错误示例（非原子删除锁）**：
```go
// 错误：非原子 GET + DEL
if val, _ := s.rdb.Get(ctx, key).Result(); val == myValue {
    s.rdb.Del(ctx, key)  // 这里 GET 到 DEL 之间锁可能已被别人持有
}
```

**正确**：
```go
// Lua: 原子比较并删除
const deleteScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`
```

### 8.10 缓存防抖 (Debounce) 规范

对于推理事件告警、状态变更通知等高频触发场景，使用 Redis `SETNX` 实现防抖：

- Key 设计：`debounce:<业务类型>:<唯一标识>`
- TTL 等于防抖窗口
- `SETNX` 返回 false 表示窗口内已触发过，丢弃本次事件
- 必须处理 Redis 故障：fail-open（允许事件通过）而非静默丢弃

### 8.11 禁止模式

- 在 for 循环中对每个元素执行缓存操作（批量读取用 `MGET` 或 `ZMSCORE`）
- 不设置 TTL 的缓存 Key
- 将事务性数据（需强一致性）放入缓存
- 以缓存数据为准而不与数据库核对
- 缓存 Key 写死字符串字面量而非常量
- MemoryCache 的 `Stop()` 未调用导致 goroutine 泄漏
- 在热路径上 `json.Marshal`/`json.Unmarshal` 大对象（考虑 `sync.Map` 或 struct 级缓存，而不是 JSON 字节级）
