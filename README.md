# Niko Admin

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-v1.10-blue?logo=go)](https://gin-gonic.com/)
[![GORM](https://img.shields.io/badge/GORM-v1.26-5c6bc0?logo=go)](https://gorm.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-✓-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![React Admin](https://img.shields.io/badge/React%20Admin-latest-4FC08D?logo=react)](https://marmelab.com/react-admin/)
[![Swagger](https://img.shields.io/badge/Swagger-✓-85EA2D?logo=swagger&logoColor=black)](https://swagger.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow)](./LICENSE)
[![PRD](https://img.shields.io/badge/PRD-✓-blue)](./docs/PRD.md)

基于 **Gin + GORM + PostgreSQL + Redis** 的后台管理系统脚手架，前端采用 **React Admin**，提供开箱即用的后台管理模板。

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| Web 框架 | [Gin](https://gin-gonic.com/) | v1.10+ |
| ORM | [GORM](https://gorm.io/) | v1.26+ |
| 数据库 | PostgreSQL | 16+ |
| 缓存 | Redis | 7+ |
| 任务队列 | [Asynq](https://github.com/hibiken/asynq) | v0.25+ |
| 认证 | JWT (golang-jwt) | v5 |
| WebSocket | gorilla/websocket | v1.5+ |
| API 文档 | [swaggo/swag](https://github.com/swaggo/swag) | v1.16+ |
| 热重载 | [air](https://github.com/air-verse/air) | v1.51+ |
| 前端 | [React Admin](https://marmelab.com/react-admin/) | latest |

## 功能

- **用户认证** — JWT 双 Token 机制（Access + Refresh），支持 Rotation 刷新与复用检测
- **RBAC 权限** — 用户/角色/权限多对多管理，中间件层拦截，Redis 缓存
- **审计日志** — 自动记录写操作，异步队列写入，不可删除
- **文件管理** — 分片上传/断点续传/多线程下载/秒传检测，支持本地/PG lo/MinIO S3 三种后端
- **国际化** — 后端 API 错误消息多语言（中/英）
- **WebSocket** — 实时通知推送，JWT 鉴权
- **异步任务** — Asynq 任务队列，支持创建/重试/取消/状态追踪
- **CRUD 代码生成** — 从 GORM Model 自动生成 Handler/Service/Repository/Router/DTO
- **Swagger 文档** — 注解驱动，自动生成，代码生成器联动
- **热重载** — air 开发模式，修改即生效

## 快速开始

### 前置条件

- Go 1.23+
- Docker & Docker Compose
- [air](https://github.com/air-verse/air) (`go install github.com/air-verse/air@latest`)
- [swag](https://github.com/swaggo/swag) (`go install github.com/swaggo/swag/cmd/swag@latest`)
- Make

### 一键初始化

```bash
git clone <repo-url> niko-admin
cd niko-admin/app
make init
```

### 启动开发

```bash
cd app
# 启动 PostgreSQL + Redis + 热重载服务
make serve

# Swagger UI 访问
# http://localhost:8080/swagger/index.html
```

### 常用命令

```bash
make serve        # 开发模式（依赖 + 热重载）
make run          # 本地运行
make build        # 编译二进制
make build-linux  # 交叉编译 Linux amd64
make swag         # 生成 Swagger 文档
make gen          # 代码生成（CRUD）
make unit-test    # 运行测试
make lint         # 代码规范检查
make migrate      # 执行数据库迁移
make docker-build # 构建 Docker 镜像
make docker-up    # 启动容器服务
make docker-down  # 停止容器服务
make clean        # 清理构建产物
make help         # 查看所有命令
```

## 项目结构

```
niko-admin/
├── app/                   # Go 后端
│   ├── cmd/
│   │   ├── server/        # 服务启动入口
│   │   ├── migrate/       # 数据库迁移
│   │   └── gen/           # 代码生成 CLI
│   ├── internal/
│   │   ├── config/        # 配置定义 & 加载 (Viper)
│   │   ├── middleware/    # Gin 中间件 (Auth/RBAC/CORS/i18n)
│   │   ├── router/        # 路由注册
│   │   ├── handler/       # HTTP Handler（REST 接口）
│   │   ├── service/       # 业务逻辑层
│   │   ├── repository/    # 数据访问层 (GORM)
│   │   ├── model/         # GORM Model 定义
│   │   ├── dto/           # 请求/响应 DTO
│   │   ├── pkg/           # 内部工具 (jwt/hash/response/errors)
│   │   └── task/          # 异步任务处理
│   ├── pkg/
│   │   ├── gen/           # CRUD 代码生成器
│   │   └── storage/       # 文件存储抽象 (local/pg/oss)
│   ├── configs/           # 多环境 YAML 配置
│   ├── docs/              # Swagger 生成文件
│   ├── Makefile
│   ├── Dockerfile
│   ├── go.mod
│   └── .air.toml
├── web/                   # 前端 (React Admin)
├── docs/
│   └── PRD.md             # 产品需求文档
├── openspec/              # OpenSpec 变更提案
├── .claude/               # AI 规范与技能
├── .opencode/             # opencode 配置
├── docker-compose.yml
└── tmp/                   # air 临时文件
```

## API 规范

### 统一响应格式

```json
// 成功
{ "code": 0, "message": "success", "data": {} }

// 错误
{ "code": 10001, "message": "用户名或密码错误" }

// 分页
{ "code": 0, "message": "success", "data": { "list": [], "total": 100, "page": 1, "page_size": 20 } }
```

### 核心接口

| 模块 | 方法 | 路径 | 说明 |
|------|------|------|------|
| Auth | POST | /api/v1/auth/login | 登录 |
| Auth | POST | /api/v1/auth/refresh | 刷新 Token |
| Auth | POST | /api/v1/auth/logout | 退出 |
| Auth | GET | /api/v1/auth/me | 当前用户 |
| Users | CRUD | /api/v1/users | 用户管理 |
| Roles | CRUD | /api/v1/roles | 角色管理 |
| Permissions | GET/POST | /api/v1/permissions | 权限管理 |
| Files | POST/GET | /api/v1/files | 文件上传管理 |
| Audit Logs | GET | /api/v1/audit-logs | 审计日志 |
| Tasks | CRUD | /api/v1/tasks | 异步任务 |
| WS | - | /api/v1/ws | WebSocket 连接 |

## 代码生成

从 GORM Model 一键生成 CRUD 代码骨架：

```bash
cd app

# 单个 Model
niko-admin gen user            # 从 internal/model/user.go 生成

# 批量所有 Model
niko-admin gen --all

# 查看可生成列表
niko-admin gen --list
```

生成内容：Handler（含 Swagger 注解）、Service、Repository、Router、DTO。

## 配置

**加载优先级**：环境变量 > `.env` 文件 > `config.dev.yaml` > 代码默认值

复制 `app/.env.example` 为 `app/.env` 按需修改：

```bash
NIKO_APP_ENV=dev
NIKO_APP_PORT=8080
NIKO_DB_HOST=localhost
NIKO_DB_PORT=5432
NIKO_DB_USER=postgres
NIKO_DB_PASSWORD=postgres
NIKO_DB_NAME=niko_admin
NIKO_REDIS_HOST=localhost
NIKO_REDIS_PORT=6379
NIKO_JWT_SECRET=change-me-in-production
NIKO_STORAGE_DRIVER=local
```

环境变量统一 `NIKO_` 前缀，下划线对应嵌套配置（如 `NIKO_DB_HOST` → `db.host`）。生产环境通过 K8s/Docker 注入，`.env` 仅用于本地开发。

## 文档

- [PRD 产品需求文档](./docs/PRD.md)
- Swagger UI: `http://localhost:8080/swagger/index.html`（开发环境）

## License

MIT
