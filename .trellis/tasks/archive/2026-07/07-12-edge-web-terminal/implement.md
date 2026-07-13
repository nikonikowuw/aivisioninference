# Web 终端 — 执行计划

## 执行阶段概览

```
Phase 1: 数据层
  Step 1  → EdgeNode 模型扩展 + Crypto 工具
  Step 2  → TerminalSession 模型 + Repository

Phase 2: 服务层
  Step 3  → SSH 连接池
  Step 4  → I/O Recorder + Asynq flush task
  Step 5  → TerminalSessionService

Phase 3: 路由层
  Step 6  → Terminal WebSocket Handler
  Step 7  → Session REST Handler + Playback Handler
  Step 8  → 路由注册 + Wire DI

Phase 4: 前端
  Step 9  → 前端 API 层 + hooks
  Step 10 → TerminalView 组件（xterm.js）
  Step 11 → TerminalTab 集成到边缘节点详情页
  Step 12 → 回放页面

Phase 5: 权限 & 清理任务
  Step 13 → 权限种子 + 中间件
  Step 14 → Asynq 清理任务 + Keepalive
```

## 验证命令

```bash
# 每次 wire 变更后
cd app && make wire

# 每次修改后验证
cd app && make build        # 编译检查
cd app && make unit-test    # 单元测试

# 前端
cd web && npm run build     # TypeScript + Vite 构建
```

## 实施步骤

### Phase 1: 数据层

#### Step 1: EdgeNode 模型扩展 + Crypto 工具

**描述：** EdgeNode struct 增加 SSH 配置字段；实现 AES-256-GCM 加解密工具函数。

**文件清单（新建）：**
- `app/internal/pkg/crypto/ssh_key.go` — `EncryptSSHKey()` / `DecryptSSHKey()` 工具函数

**文件清单（修改）：**
- `app/internal/model/edge_node.go` — 追加 `SSHPort int` + `SSHPrivateKey string` 字段

**验证：** `cd app && make build`

#### Step 2: TerminalSession 模型 + Repository

**描述：** 新建 TerminalSession 数据模型和对应的 Repository。

**文件清单（新建）：**
- `app/internal/model/terminal_session.go` — TerminalSession struct + 状态常量
- `app/internal/repository/terminal_session.go` — CRUD + ListByNodeID + ListPaused + CountActiveByNode

**验证：** `cd app && make build`

### Phase 2: 服务层

#### Step 3: SSH 连接池

**描述：** 实现 SSHPool，管理按 nodeID 池化的 ssh.Client，支持引用计数和 idle 超时。

**文件清单（新建）：**
- `app/internal/service/ssh_pool.go` — SSHPool + SSHClientEntry 实现

**核心接口：**
```
AcquireClient(ctx, nodeID, endpoint, sshPort, encryptedKey) → *SSHClientEntry, error
ReleaseClient(nodeID)
AcquireSession(ctx, nodeID, sessChKey, cols, rows) → *ssh.Session, error
ReleaseSession(nodeID, sessChKey)
```

**验证：** `cd app && make build`

#### Step 4: I/O Recorder + Asynq flush task

**描述：** Recorder 实时录制 ttyrec 格式；Asynq task 异步持久化到 DB。

**文件清单（新建）：**
- `app/internal/service/recorder.go` — Recorder + ttyRecorder 结构
- `app/internal/task/terminal_session.go` — `TypeTerminalSessionRecordFlush` task handler

**验证：** `cd app && make build`

#### Step 5: TerminalSessionService

**描述：** 核心业务逻辑：创建/恢复/暂停/关闭会话、录制管理。

**文件清单（新建）：**
- `app/internal/service/terminal_session.go` — TerminalSessionService

**核心方法：**
```
CreateSession(ctx, userID, nodeID, cols, rows) → *TerminalSession, error
ResumeSession(ctx, sessionID, userID, cols, rows) → *TerminalSession, error
PauseSession(ctx, sessionID, userID) error
CloseSession(ctx, sessionID, userID, reason) error
ListSessions(ctx, nodeID, status, page, pageSize) ([]TerminalSession, int64, error)
GetSessionRecording(ctx, sessionID) ([]byte, error)
ForceCloseSession(ctx, sessionID, userID) error
```

**验证：** `cd app && make build`

### Phase 3: 路由层

#### Step 6: Terminal WebSocket Handler

**描述：** 核心 WebSocket handler，负责连接生命周期管理、消息协议解析和 SSH I/O 桥接。

**文件清单（新建）：**
- `app/internal/handler/terminal.go` — TerminalHandler

**关键流程：**
```
HandleWebSocket(c *gin.Context):
  1. JWT 认证 + edge:terminal 权限校验
  2. 解析 query param: node_id, session_id(可选), cols(默认80), rows(默认24)
  3. session_id 为空 → svc.CreateSession → 新 SSH 连接
     session_id 非空 → svc.ResumeSession → 恢复 SSH I/O
  4. ws.Upgrader.Upgrade → 建立 WebSocket
  5. 启动 Recorder.StartRecording
  6. goroutine: SSH stdout → WebSocket output msg
  7. goroutine: WebSocket input/resize msg → SSH stdin/window-change
  8. 断开连接 → svc.PauseSession → 5min 超时倒计时
  9. WebSocket 关闭/错误 → svc.CloseSession
```

**验证：** `cd app && make build`

#### Step 7: Session REST Handler + Playback Handler

**描述：** 会话管理的 REST API 和录制回放 API。

**文件清单（新建）：**
- `app/internal/handler/terminal_session.go` — SessionHandler
- `app/internal/handler/terminal_playback.go` — PlaybackHandler
- `app/internal/dto/terminal_session.go` — request/response DTO

**验证：** `cd app && make build`

#### Step 8: 路由注册 + Wire DI

**描述：** 注册 WebSocket 端点、REST 路由，Wire 注入所有新依赖。

**文件清单（修改）：**
- `app/internal/router/deps.go` — 新增 provider 函数
- `app/internal/router/wire.go` — 注册 Wire 依赖
- `app/internal/router/wire_gen.go` — `make wire` 生成
- `app/internal/router/router.go` — 注册路由
- `app/internal/server/server.go` — AutoMigrate 注册 TerminalSession 模型

**验证：** `cd app && make wire && make build`

### Phase 4: 前端

#### Step 9: 前端 API 层 + hooks

**描述：** 前端终端 API 服务层和 WebSocket hook。

**文件清单（新建）：**
- `web/src/services/terminal.ts` — REST API + WebSocket URL builder
- `web/src/hooks/useTerminal.ts` — useTerminal hook（xterm 实例管理 + WS 连接）

**验证：** `cd web && npm run build`

#### Step 10: TerminalView 组件（xterm.js）

**描述：** 通用 xterm.js 渲染组件，支持连接/断开/全屏。

**文件清单（新建）：**
- `web/src/components/terminal/TerminalView.tsx` — xterm.js Terminal 渲染组件
- `web/src/components/terminal/SessionList.tsx` — 可恢复的会话列表

**依赖：**
```json
"@xterm/xterm": "^5.3.0",
"@xterm/addon-fit": "^0.8.0"
```

**验证：** `cd web && npm install && npm run build`

#### Step 11: TerminalTab 集成到边缘节点详情页

**描述：** 在边缘节点详情页增加"终端"Tab。

**文件清单（新建）：**
- `web/src/views/admin/devices/edge-nodes/components/TerminalTab.tsx`

**文件清单（修改）：**
- `web/src/views/admin/devices/edge-nodes/detail.tsx` — 添加 Tab，引入 TerminalTab

**操作流程：**
1. 节点 offline 或未配置 SSH Key → 按钮置灰，tooltip 提示
2. 点击"打开终端"→ 创建 session → WebSocket 连接 → xterm 渲染
3. 断开连接 → session paused，提示"会话已暂停，可在 5 分钟内恢复"
4. 点击"恢复"→ 展示可恢复会话列表 → 选择 → Resume

**验证：** `cd web && npm run build`

#### Step 12: 回放页面

**描述：** ttyrec 录制回放页面，支持播放/暂停/跳转。

**文件清单（新建）：**
- `web/src/components/terminal/PlaybackView.tsx` — 回放组件
- `web/src/views/admin/devices/edge-nodes/playback.tsx` — 独立回放页面

**验证：** `cd web && npm run build`

### Phase 5: 权限 & 清理任务

#### Step 13: 权限种子 + 中间件

**描述：** 注册新权限代码 `edge:terminal` 和 `edge:terminal:audit`；在前端页面路由中加入权限校验。

**文件清单（修改）：**
- `app/internal/service/permission.go` 或类似位置 — 权限种子注册
- `web/src/router/` — 回放页面路由添加权限校验

**验证：** `cd app && make build && cd web && npm run build`

#### Step 14: Asynq 清理任务 + Keepalive

**描述：** 注册 TerminalSession cleanup task（`*/5 * * * *`）和 SSH 连接池 Keepalive。

**文件清单（修改）：**
- `app/internal/task/terminal_session.go` — 追加 `TerminalSessionCleanupTask` + `TerminalSessionHeartbeatTask`
- `app/internal/router/router.go` — 注册到 Asynq scheduler

**验证：** `cd app && make build && make unit-test`

## 风险点 / 回滚点

| 步骤 | 风险 | 回滚操作 |
|------|------|---------|
| Step 1 | SSH 密钥加密配置有误导致数据不可逆 | 删除加密字段，退化到无 SSH 配置 |
| Step 3 | SSH 连接池资源泄漏 | 缩减 idle timeout，增加 max conn 限制 |
| Step 6 | WebSocket 消息协议设计不合理 | 前端/后端同步改协议类型名即可，不影响数据层 |
| Step 8 | Wire DI 循环依赖 | 解耦 provider，必要时分步注册 |
| Step 11 | xterm.js 与现有 Chakra UI 样式冲突 | CSS scoped 隔离，降到 xterm.css 默认主题 |
| Step 14 | Asynq 清理误删活跃会话 | 先以 info 级别日志观察 2 个周期再开启删除 |

## 未完成项（已知且可接受）

- SSH 密钥的自动化分发到边缘节点 — 需管理端手动配置私钥
- 多用户协作查看同一会话 — 后续版本
- 文件上传/下载（scp/sftp）— 后续版本
- 默认 SSH 用户配置（当前硬编码为 `admin`）

## 前置检查清单（task.py start 前）

- [ ] PRD 已确认并通过用户 review
- [ ] Design 文档已完成
- [ ] 以上实施步骤的依赖关系已理清
- [ ] 新增文件的命名约定已最终确定
- [ ] frontend xterm.js 依赖已确认版本兼容性
