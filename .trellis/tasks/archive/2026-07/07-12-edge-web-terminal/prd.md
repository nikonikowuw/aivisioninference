# Web 终端：边缘节点远程交互式 Shell

## Goal

为管理员提供浏览器端的交互式 Shell 访问，实现对边缘节点的远程运维操作。Go 后端通过 SSH 代理连接到边缘节点，WebSocket 将终端 I/O 实时传输到浏览器，并提供完整的会话管理和审计录制能力。

## Design Decisions

| 决策 | 方案 | 理由 |
|------|------|------|
| 通道协议 | SSH Proxy（Go `crypto/ssh`） | 边缘节点为标准 Linux，SSH 标配；C++ 引擎侧零改动 |
| 认证方式 | SSH Private Key（AES-256 加密存 DB） | 密钥认证安全，管理 UI 配置私钥，无密码泄露风险 |
| 会话拓扑 | 有状态会话管理 | WebSocket 断连后会话可暂停/恢复；支持 session 共享查看 |
| SSH 连接复用 | `ssh.Client` 按 `nodeID` 池化，每个 Session 独立 `ssh.Session` | 减少 TCP 握手，不影响 shell 隔离性 |
| 审计录制 | 全 I/O 录制（ttyrec 格式）异步写入 PostgreSQL | 运维审计基线，Asynq 异步不阻塞实时 I/O |

## Requirements

### 1. SSH 密钥管理

- `EdgeNode` 模型增加 `ssh_port`（int，默认 22）和 `ssh_private_key`（text，AES-256 加密存储）字段
- 管理端边缘节点编辑/创建页增加 SSH 配置区域：端口 + 私钥（textarea，粘贴 PEM 格式）
- 私钥入库前用 AES-256-GCM 加密，应用层透明加解密
- 后端启动时按需预连接到在线节点的 SSH（可选预热）

### 2. 终端会话管理

- 会话模型 `TerminalSession` 记录会话元数据：
  - `id`, `node_id`, `user_id`, `user_name`
  - `status`（active / paused / closed）
  - `started_at`, `ended_at`, `duration_seconds`
  - `recording_path`（ttyrec 录制文件在 OSS 或 DB 的引用）
  - `reason`（closed / timeout / error）
- 提供 REST API 列出现有会话（`GET /edge-nodes/:id/sessions`），支持按节点/状态/用户筛选
- 管理员可强制关闭某个会话（`DELETE /edge-nodes/:id/sessions/:session_id`）

### 3. 实时终端（WebSocket）

- 新 WebSocket 端点：`/ws/terminal?token=<jwt>&node_id=<id>&session_id=<id>`
  - `session_id` 可选：不传则创建新会话，传则恢复已有暂停的会话
- 协议：纯 JSON 消息封装
  - 服务器 → 客户端：`{"type":"output","data":"...","timestamp":...}`
  - 客户端 → 服务器：`{"type":"input","data":"..."}`
  - 服务器 → 客户端：`{"type":"resize","cols":80,"rows":24}`（初始化时下发终端尺寸）
  - 客户端 → 服务器：`{"type":"resize","cols":80,"rows":24}`（窗口改变时通知）
  - 服务器 → 客户端：`{"type":"paused","reason":"..."}` / `{"type":"resumed"}`
  - 服务器 → 客户端：`{"type":"closed","reason":"..."}`
- SSH pty 尺寸与前端 xterm.js 尺寸同步（resize 事件）
- 免交互 sudo 支持：SSH 以 `admin`（或配置用户）登录，`sudo` 通过 `/etc/sudoers` 预先配置无需交互提权

### 4. 会话暂停/恢复

- WebSocket 断开时，SSH Session 不断开，转为 `paused` 状态
- SSH Session 保持 5 分钟超时，超时未恢复则关闭
- 同一节点的同用户可以通过新的 WebSocket 连接恢复暂停的会话
- 管理 UI 展示暂停中的会话列表，管理员可 "重新连接"

### 5. I/O 录制与回放

- 全双工录制格式为 ttyrec（时间戳 + 原始字节），存储为 JSONB 或分段存储在 PostgreSQL（或小型文件存 OSS）
- 使用 Asynq task 异步写入，不阻塞实时终端 I/O
- 录制包含完整的用户输入和远程输出
- 管理端提供 Session 回放页面：可播放/暂停/跳转到任意时间点的终端输出

### 6. 安全与权限

- 新增权限代码：`edge:terminal`（控制终端访问能力）
- WebSocket 端点校验 JWT 且校验 `edge:terminal` 权限
- API 操作均受 RBAC 控制
- SSH 私钥在 API 响应中永不暴露（`json:"-"`）
- 会话录制仅允许有 `edge:terminal:audit` 权限的管理员查看

### 7. 前端终端页面

- 边缘节点详情页增加 "终端" Tab（使用 xterm.js 渲染）
- Tab 中的操作按钮：连接、断开、全屏
- 会话恢复时展示可恢复的会话列表供选择
- 回放页面：独立页面或 Modal，带播放控制条

## Acceptance Criteria

- [ ] `EdgeNode` 表新增 `ssh_port` 和 `ssh_private_key` 字段，CURD API 支持 SSH 配置，私钥加密存储且永不暴露
- [ ] 新的 WebSocket 端点 `/ws/terminal` 支持创建、暂停、恢复、关闭终端会话
- [ ] 支持终端尺寸同步（resize），xterm.js 显示正确且交互流畅
- [ ] 支持会话暂停（WebSocket 断连后 5 分钟内可恢复）
- [ ] 完整的 ttyrec 录制 + Asynq 异步写入 + 管理端回放查看
- [ ] RBAC 权限 `edge:terminal` 正确拦截未授权访问
- [ ] `GET /edge-nodes/:id/sessions` / `DELETE /edge-nodes/:id/sessions/:id` 可用
- [ ] 前端边缘节点详情页新增终端 Tab，可连接/断开/全屏/恢复会话
- [ ] 前端会话回放页可用

## Out of Scope

- 文件上传/下载（scp/sftp）— 后续版本
- 多用户同时共享同一 Session 的协作查看 — 后续版本
- SSH 密钥自动分发到边缘节点的机制 — 后续版本
- 端口转发（SSH tunnel）— 后续版本
- 非 SSH 协议（如串口）支持 — 后续版本
