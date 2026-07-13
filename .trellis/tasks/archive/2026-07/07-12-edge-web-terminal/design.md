# Web 终端 — 技术设计方案

## 1. 架构总览

```
┌──────────────────────────────────────────────────────────────────┐
│                        浏览器 (xterm.js)                          │
│   ┌────────────────────────────────────────────────────────┐     │
│   │  WebSocket: wss://host/ws/terminal?token=<jwt>&node=XX │     │
│   │  协议: JSON 消息 (input / output / resize / control)   │     │
│   └────────────────────────┬───────────────────────────────┘     │
└────────────────────────────┼─────────────────────────────────────┘
                             │
┌────────────────────────────▼─────────────────────────────────────┐
│                    Go 控制面 (app/internal/)                      │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ TerminalHandler  │  │ SessionHandler   │  │ PlaybackHandler │ │
│  │ (WebSocket)      │  │ (REST: CRUD)     │  │ (REST: 回放)    │ │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬────────┘ │
│           │                     │                      │          │
│  ┌────────▼─────────────────────▼──────────────────────▼────────┐ │
│  │                TerminalSessionService                          │ │
│  │  - CreateSession / ResumeSession / PauseSession / CloseSession │ │
│  │  - ListSessions / GetSessionRecording                          │ │
│  └────────────────────────┬──────────────────────────────────────┘ │
│                           │                                        │
│  ┌────────────────────────▼──────────────────────────────────────┐ │
│  │              SSH Connection Pool (按 nodeID 池化)              │ │
│  │  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐    │ │
│  │  │ ssh.Client     │ │ ssh.Client     │ │ ssh.Client     │    │ │
│  │  │ nodeID=A       │ │ nodeID=B       │ │ nodeID=C       │    │ │
│  │  │ session1(dev1) │ │ session5(dev2) │ │                │    │ │
│  │  │ session2(dev1) │ │                │ │                │    │ │
│  │  └────────────────┘ └────────────────┘ └────────────────┘    │ │
│  └────────────────────────┬──────────────────────────────────────┘ │
│                           │                                        │
│  ┌────────────────────────▼──────────────────────────────────────┐ │
│  │              Session I/O Recorder (Asynq)                     │ │
│  │  ttyrec 格式 → 分段 JSONB 存储 → Asynq 异步写入 PostgreSQL     │ │
│  └───────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
                             │
                 SSH (端口 22, 默认)
                             │
┌────────────────────────────▼─────────────────────────────────────┐
│                  边缘节点 (Linux)                                 │
│  - SSHD running                                                │
│  - admin 用户 (sudo 免密配置)                                    │
│  - C++ Engine 无需任何修改                                        │
└──────────────────────────────────────────────────────────────────┘
```

## 2. 新增/修改数据模型

### 2.1 EdgeNode 扩展字段

```go
// 在现有 EdgeNode struct 中追加：
SSHPort        int    `gorm:"default:22;comment:SSH 端口" json:"ssh_port"`
SSHPrivateKey  string `gorm:"type:text;comment:SSH 私钥(AES-256-GCM 加密)" json:"-"`
```

- `SSHPrivateKey` 入库前使用 AES-256-GCM 加密，应用层通过 `crypto/aes` + `crypto/cipher` 加解密
- API 响应中 `json:"-"` 确保永不暴露
- `SSHPort` 默认 22，管理员可在编辑页修改

### 2.2 TerminalSession 终端会话模型

```go
type TerminalSession struct {
    BaseModel

    NodeID          string     `gorm:"type:uuid;not null;index;comment:边缘节点ID"`
    UserID          string     `gorm:"type:uuid;not null;index;comment:用户ID"`
    UserName        string     `gorm:"type:varchar(100);not null;comment:用户名"`
    Status          string     `gorm:"type:varchar(20);not null;default:active;comment:状态(active/paused/closed)"`
    Reason          string     `gorm:"type:varchar(50);comment:关闭原因(closed/timeout/error)"`
    ErrorMessage    string     `gorm:"type:text;comment:错误信息"`
    StartedAt       time.Time  `gorm:"not null;comment:会话开始时间"`
    EndedAt         *time.Time `gorm:"comment:会话结束时间"`
    DurationSeconds int        `gorm:"comment:持续时长(秒)"`
    RecordingData   []byte     `gorm:"type:jsonb;comment:ttyrec录制数据(JSONB)"`
    SessChKey       string     `gorm:"type:varchar(64);uniqueIndex;comment:SSH Session通道key(连接池索引)"`
    PausedAt        *time.Time `gorm:"comment:暂停时间"`

    Node *EdgeNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

const (
    TerminalSessionStatusActive  = "active"
    TerminalSessionStatusPaused  = "paused"
    TerminalSessionStatusClosed  = "closed"
)
```

**关键设计说明：**
- `RecordingData` 使用 JSONB 存储 ttyrec 格式：`[{"time": <秒.微秒>, "data": "<base64>"}, ...]`
- 实时 I/O 在 WebSocket 层写入 channel 缓冲，Recorder goroutine 消费并定期（每 5 秒或每 100 条）通过 Asynq task 异步持久化
- `SessChKey` 用于 SSH Client 池中定位具体的 `ssh.Session` 实例（格式：`{nodeID}:{traceID}`）
- 录制数据量控制：单个 session 超过 10MB 时截断并记录 warning 日志

### 2.3 权限种子

在权限初始化代码中新增：

```go
// Permission code
EdgeTerminalPermission      = "edge:terminal"       // Web 终端访问
EdgeTerminalAuditPermission = "edge:terminal:audit"  // 终端回放审计
```

## 3. 分层组件

### 3.1 Repository 层

| Repository | 新增方法 |
|---|---|
| `TerminalSessionRepository` | CRUD, `ListByNodeID`, `ListByUserID`, `ListPaused()`, `CountActiveByNodeID` |
| `EdgeNodeRepository`（修改） | 无需新方法，模型字段变更通过 GORM AutoMigrate 自动处理 |

### 3.2 Service 层

**SSH Connection Pool（`ssh_pool.go`）**

```go
type SSHClientEntry struct {
    Client    *ssh.Client
    NodeID    string
    ConnectedAt time.Time
    mu        sync.Mutex
    sessions  map[string]*ssh.Session  // key = sessChKey
}

type SSHPool struct {
    mu      sync.RWMutex
    clients map[string]*SSHClientEntry  // key = nodeID
    logger  *zap.Logger
}

func (p *SSHPool) AcquireClient(ctx context.Context, nodeID string, sshHost string, sshPort int, signer ssh.Signer) (*SSHClientEntry, error)
func (p *SSHPool) ReleaseClient(nodeID string)
func (p *SSHPool) AcquireSession(ctx context.Context, nodeID string, sessChKey string, cols, rows int) (*ssh.Session, error)
func (p *SSHPool) ReleaseSession(nodeID string, sessChKey string)
func (p *SSHPool) Keepalive(ctx context.Context)  // 定期心跳
```

**池化策略：**
- `ssh.Client` 按 `nodeID` 池化，引用计数管理
- 引用计数归零后保持 5 分钟 idle 超时再关闭（避免频繁建连）
- 每个 Session 在 Client 上调用 `client.NewSession()`，通过 `sessChKey` 映射管理
- `crypto/ssh` 的 Client 本身支持 Session 复用（同一条 TCP 连接），因此多个 Session 共享一个 Client 无冲突

**TerminalSessionService（`terminal_session.go`）**

| 方法 | 职责 |
|---|---|
| `CreateSession(ctx, userID, nodeID, cols, rows) (*TerminalSession, error)` | 创建新会话，建立 SSH 连接 |
| `ResumeSession(ctx, sessionID, userID, cols, rows) (*TerminalSession, error)` | 恢复暂停会话（同一用户轻校验） |
| `PauseSession(ctx, sessionID, userID) error` | 暂停（WebSocket 断连时触发） |
| `CloseSession(ctx, sessionID, userID, reason) error` | 关闭会话，释放 SSH Session |
| `ListSessions(ctx, nodeID, status, page, pageSize) ([]TerminalSession, int64, error)` | 分页列会话 |
| `GetSessionRecording(ctx, sessionID) ([]byte, error)` | 获取 ttyrec 录制数据 |
| `ForceCloseSession(ctx, sessionID, userID) error` | 管理员强制关闭 |

**Recorder（`recorder.go`）**

```go
type Recorder struct {
    sessions sync.Map  // sessionID → *ttyRecorder
}

type ttyRecorder struct {
    mu          sync.Mutex
    sessionID   string
    entries     []ttyrecEntry
    lastFlush   time.Time
    bufferSize  int
}

type ttyrecEntry struct {
    Time  float64 `json:"time"`  // 会话内相对时间（秒）
    Data  string  `json:"data"`  // base64 编码的字节
}

func (r *Recorder) StartRecording(sessionID string)
func (r *Recorder) Record(sessionID string, data []byte)  // 异步追加到 buffer
func (r *Recorder) Flush(sessionID string)                 // 强制刷新到 DB
func (r *Recorder) StopRecording(sessionID string)         // 最后一次 flush + 清理
```

**I/O 录制策略：**
- `Record()` 写入 ring buffer，不阻塞 SSH I/O 路径
- 每 5 秒或 buffer 超过 100 条时自动 flush
- Flush 通过 `Asynq` task 异步写入 PostgreSQL JSONB（避免 WebSocket 路径阻塞 DB 写）
- 使用 Asynq 的 `task.NewEdgeTerminalRecordFlushTask(sessionID, entries)` 入队

### 3.3 Handler 层

**TerminalHandler — WebSocket 终端（`terminal_handler.go`）**

```
WebSocket: GET /ws/terminal?token=<jwt>&node_id=<id>[&session_id=<id>&cols=80&rows=24]
```

**连接生命周期：**

```
1. JWT 认证 + 校验 edge:terminal 权限
2. session_id 为空 → 创建新 TerminalSession（SSH 连接）
   session_id 非空 → 校验该 session 属于该用户 & 节点 → Resume
3. 启动 Recorder
4. 建立 SSH pty（cols/rows 来自 query 参数）
5. WebSocket read goroutine: input → SSH stdin
6. WebSocket read goroutine: resize → SSH window-change
7. SSH stdout goroutine: output → WebSocket write
8. WebSocket disconnect → PauseSession（5min 超时后 CloseSession）
9. WebSocket close / error → CloseSession
```

**消息协议（详尽的类型枚举）：**

| Direction | Type | Payload | Notes |
|---|---|---|---|
| S→C | `output` | `{"data":"..."}` | SSH stdout 输出 |
| C→S | `input` | `{"data":"..."}` | 用户键盘输入 |
| C→S | `resize` | `{"cols":80,"rows":24}` | 窗口尺寸变更 |
| S→C | `resize` | `{"cols":80,"rows":24}` | 初始终端尺寸通知 |
| S→C | `paused` | `{"reason":"disconnect"}` | WebSocket 断连，会话暂停 |
| S→C | `resumed` | `{"session_id":"..."}` | 会话恢复 |
| S→C | `closed` | `{"reason":"timeout"}` | 会话终止 |
| S→C | `heartbeat` | `{}` | 每 30 秒 keepalive |

**SessionHandler — REST（`session_handler.go`）**

| Method | Path | 说明 |
|---|---|---|
| GET | `/edge-nodes/:id/sessions` | 列节点会话（支持 status 过滤） |
| GET | `/edge-nodes/:id/sessions/:sessionId` | 会话详情 |
| DELETE | `/edge-nodes/:id/sessions/:sessionId` | 强制关闭 |
| GET | `/edge-nodes/:id/sessions/:sessionId/recording` | 获取录制数据 |

**PlaybackHandler — 回放（`playback_handler.go`）**

| Method | Path | 说明 |
|---|---|---|
| GET | `/terminal/sessions/:sessionId/playback` | 返回 ttyrec JSON 数据 |
| GET | `/terminal/playback-page/:sessionId` | 回放页面 SPA 入口（可选） |

### 3.4 Asynq Task 层

| Task | 周期 | 职责 |
|---|---|---|
| `TypeTerminalSessionRecordFlush` | 按需 | 将 Recorder buffer 写入 DB |
| `TerminalSessionCleanupTask` | `*/5 * * * *` | 检查 paused > 5min 的 session，强制关闭 |
| `TerminalSessionHeartbeatTask` | `* * * * *` | SSH 连接池 keepalive，检测异常连接 |

### 3.5 前端组件

| 组件 | 路径 | 说明 |
|---|---|---|
| `TerminalTab` | `web/src/views/admin/devices/edge-nodes/components/TerminalTab.tsx` | 边缘节点详情页的终端 Tab |
| `TerminalView` | `web/src/components/terminal/TerminalView.tsx` | 通用 xterm.js 渲染组件（可复用） |
| `SessionList` | `web/src/components/terminal/SessionList.tsx` | 恢复会话列表 |
| `PlaybackView` | `web/src/components/terminal/PlaybackView.tsx` | ttyrec 回放组件 |
| `PlaybackPage` | `web/src/views/admin/devices/edge-nodes/playback.tsx` | 独立回放页面 |

**前端 API 层（`web/src/services/terminal.ts`）：**

```typescript
export const terminalApi = {
  listSessions: (nodeId: string, params?: { status?: string }) => ...
  getSession: (nodeId: string, sessionId: string) => ...
  closeSession: (nodeId: string, sessionId: string) => ...
  getRecording: (sessionId: string) => ...
  getPlaybackURL: (sessionId: string) => string  // WebSocket URL builder
}
```

**WebSocket 连接封装（`web/src/hooks/useTerminal.ts`）：**

```typescript
export function useTerminal(nodeID: string, sessionID?: string) {
  // 返回: { connect, disconnect, sendInput, sendResize, status, session }
  // 内部: 使用 xterm.js Terminal 实例 + WebSocket
}
```

## 4. 路由注册

### 4.1 后端路由

```go
// 终端 WebSocket (认证 + 权限前置)
v1.GET("/ws/terminal", rbac.Check("edge:terminal"), terminalHandler.HandleWebSocket)

// 终端 REST API (RBAC 保护)
authorized := authorized.Group("/edge-nodes")
authorized.Use(r.RBAC())
{
    // 终端会话管理
    authorized.GET("/:id/sessions", terminalHandler.ListSessions)
    authorized.GET("/:id/sessions/:sessionId", terminalHandler.GetSession)
    authorized.DELETE("/:id/sessions/:sessionId", terminalHandler.CloseSession)

    // 录制回放 (edge:terminal:audit)
    authorized.GET("/:id/sessions/:sessionId/recording", rbac.Check("edge:terminal:audit"), terminalHandler.GetRecording)
}
```

### 4.2 前端路由

```typescript
// 回放页面
{ path: "/admin/devices/edge-nodes/playback/:sessionId", component: PlaybackPage }
```

## 5. SSH 连接池生命周期

```
AcquireClient(nodeID):
  ├─ clients[nodeID] 存在 → refCount++, return
  └─ clients[nodeID] 不存在:
       ├─ DB 查询 EdgeNode: endpoint, ssh_port, ssh_private_key
       ├─ AES-256-GCM 解密 private key
       ├─ ssh.ParsePrivateKey → signer
       ├─ ssh.Dial("tcp", host:port, sshConfig)
       ├─ 存入 clients[nodeID]
       └─ return

ReleaseSession(nodeID, sessChKey):
  ├─ clients[nodeID] 存在
  │    └─ sessions[sessChKey].Close()
  │    └─ delete sessions[sessChKey]
  └─ refCount--

ReleaseClient(nodeID):
  ├─ refCount→0, 启动 idle timeout (5min)
  ├─ idle 期间有新的 AcquireClient → 取消 idle timeout
  └─ idle 超时 → ssh.Client.Close(), delete clients[nodeID]

Keepalive → 遍历 clients, 发送 KeepAlive 请求
```

## 6. 前端 WebSocket + xterm.js 集成

### 6.1 初始化流程

```
1. xterm.js: new Terminal({ cols: 80, rows: 24, cursorBlink: true })
2. Terminal.attachCustomKeyEventHandler 处理 Ctrl+C 等
3. useTerminal hook 建立 WebSocket 连接
4. 绑定 xterm.onData → sendInput
5. 绑定 window.onresize → sendResize
6. onmessage: {
     output: terminal.write(data)
     resize: terminal.resize(cols, rows)
     paused: show overlay "会话已暂停"
     resumed: hide overlay
     closed: terminal.dispose()
     heartbeat: ignore
   }
```

### 6.2 依赖

```json
{
  "@xterm/xterm": "^5.3.0",
  "@xterm/xterm": {"version": "^5.3.0"},
  "@xterm/addon-fit": "^0.8.0",
  "@xterm/addon-web-links": "^0.8.0",
  "@xterm/addon-search": "^0.13.0"
}
```

## 7. 加密方案（SSH 私钥）

```go
const sshKeyEncryptionKey = ""  // 从环境变量中读取 ENC_KEY 或 HKDF 派生

func encryptSSHKey(plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher([]byte(encryptionKey))
    aead, _ := cipher.NewGCM(block)
    nonce := make([]byte, aead.NonceSize())
    rand.Read(nonce)
    return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptSSHKey(ciphertext []byte) ([]byte, error) {
    block, _ := aes.NewCipher([]byte(encryptionKey))
    aead, _ := cipher.NewGCM(block)
    nonceSize := aead.NonceSize()
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return aead.Open(nil, nonce, ciphertext, nil)
}
```

- 加密密钥从环境变量 `SSH_KEY_ENC_KEY` 读取（32 字节 hex）
- 每个节点使用随机 nonce，nonce 与密文一同存储（nonce + ciphertext）
- 应用层在 Service `Create/Update EdgeNode` 时自动加密，`GetEdgeNode` 时自动解密（仅用于 SSH 连接，不返回给 API）

## 8. Wire DI 新增依赖

```
repositories:
  - TerminalSessionRepository

services:
  - TerminalSessionService
  - SSHPool

handlers:
  - TerminalHandler

tasks:
  - TerminalSessionRecordFlushTask
  - TerminalSessionCleanupTask
```

Provider 定义在 `deps.go`：

```go
func provideTerminalSessionRepository(db *gorm.DB) *repository.TerminalSessionRepository
func provideSSHPool(logger *zap.Logger) *service.SSHPool
func provideTerminalSessionService(repo *repository.TerminalSessionRepository, pool *service.SSHPool, recorder *service.Recorder, rdb *redis.Client, logger *zap.Logger) *service.TerminalSessionService
func provideTerminalHandler(svc *service.TerminalSessionService, jwt *jwtutil.Manager, rbac *middleware.RBACMiddleware, logger *zap.Logger) *handler.TerminalHandler
```

## 9. WebSocket 事件通知

复用现有 `ws.Hub`，在终端 Session 状态变更时广播：

```
terminal.session.created  → { session_id, node_id, user_id, status, started_at }
terminal.session.closed   → { session_id, node_id, user_id, reason, duration }
terminal.session.error    → { session_id, node_id, error_message }
```

## 10. 迁移和回滚方案

| 操作 | 步骤 |
|---|---|
| **部署** | GORM AutoMigrate 自动创建 `edge_nodes.ssh_port/ssh_private_key` 列 + `terminal_sessions` 表 |
| **回滚** | `ALTER TABLE edge_nodes DROP COLUMN ssh_port, DROP COLUMN ssh_private_key; DROP TABLE terminal_sessions;` |
| **数据兼容** | 现有节点 `ssh_port` 默认 22，`ssh_private_key` 为空时前端禁用终端连接按钮 |
| **键轮转** | 环境变量 `SSH_KEY_ENC_KEY` 变更后，所有已存私钥失效，需管理员重新配置 |

## 11. 安全考虑

| 风险 | 缓解措施 |
|---|---|
| SSH 私钥泄露 DB | AES-256-GCM 加密存储；API 永不返回（json:"-"） |
| 未授权终端访问 | WebSocket 端点和 REST API 均受 JWT + RBAC edge:terminal 保护 |
| 会话劫持 | Resume 时校验 session 的 nodeID + userID 匹配 |
| 终端 I/O 中敏感信息泄露 | 录制数据受 edge:terminal:audit 权限保护 |
| SSH 连接长时间空闲 | 5min 会话超时强制关闭；SSH Client 层有 Keepalive |
| 前端 XSS 注入 | xterm.js 默认不做 HTML 渲染，输出原义字符 |

## 12. 性能与限制

- **单节点并发上限**：默认每个节点最多 10 个并发 Session（`max_concurrent_sessions` 节点维度的保护）
- **录制数据上限**：单个 Session 录制不超过 10MB，超过则截断并记录 warning
- **WebSocket 消息缓冲**：256 条 buffer，满时丢弃旧消息
- **SSH 连接池 idle 超时**：5 分钟后自动关闭
