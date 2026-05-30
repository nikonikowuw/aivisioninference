# Niko Admin

<p align="center">
  <strong>简体中文</strong> | <a href="./README_EN.md">English</a>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go Version"></a>
  <a href="https://gin-gonic.com/"><img src="https://img.shields.io/badge/Gin-v1.10-blue?logo=go" alt="Gin"></a>
  <a href="https://gorm.io/"><img src="https://img.shields.io/badge/GORM-v1.26-5c6bc0?logo=go" alt="GORM"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL"></a>
  <a href="https://redis.io/"><img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis"></a>
  <a href="https://www.docker.com/"><img src="https://img.shields.io/badge/Docker-✓-2496ED?logo=docker&logoColor=white" alt="Docker"></a>
  <a href="https://vite.dev/"><img src="https://img.shields.io/badge/Vite-6.x-646CFF?logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://chakra-ui.com/"><img src="https://img.shields.io/badge/Chakra--UI-2.x-319795?logo=chakra-ui&logoColor=white" alt="Chakra UI"></a>
  <a href="https://swagger.io/"><img src="https://img.shields.io/badge/Swagger-✓-85EA2D?logo=swagger&logoColor=black" alt="Swagger"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow" alt="License"></a>
  <a href="./docs/PRD.md"><img src="https://img.shields.io/badge/PRD-✓-blue" alt="PRD"></a>
</p>

基于 **Gin + GORM + PostgreSQL + Redis** 的全栈后台管理系统脚手架。后端具备健全的 RBAC、双 Token 认证、文件分片上传及异步任务处理能力；前端基于 **React + Vite + Chakra UI + TypeScript** 开发，提供现代、流畅且高颜值的响应式后台管理模版。

---

## 📖 目录

- [技术栈](#-技术栈)
- [核心功能](#-核心功能)
- [系统架构](#-系统架构)
- [快速开始](#-快速开始)
  - [前置条件](#前置条件)
  - [后端启动](#后端启动)
  - [前端启动](#前端启动)
- [🐳 Docker 部署](#-docker-部署)
  - [本地开发依赖环境](#1-本地开发依赖环境-postgres--redis)
  - [本地一键容器化运行](#2-本地一键容器化运行)
  - [生产环境部署](#3-生产环境部署-docker-composeprodyml)
- [日常开发常用命令](#-日常开发常用命令)
- [项目目录结构](#-项目目录结构)
- [API 规范与响应格式](#-api-规范与响应格式)
- [代码生成器 (CRUD)](#-代码生成器-crud)
- [配置文件说明](#-配置文件说明)
- [相关文档](#-相关文档)
- [License](#license)

---

## 🛠 技术栈

### 后端 (Go)

- **Web 框架**: [Gin v1.10+](https://gin-gonic.com/) - 高性能 HTTP 路由与中间件框架

- **ORM 框架**: [GORM v1.26+](https://gorm.io/) - 强大、对开发者友好的 Go 语言 ORM 库
- **主数据库**: PostgreSQL 16+ - 支持强一致性与复杂查询
- **高速缓存**: Redis 7+ - 缓存 RBAC 权限、会话及限流状态
- **异步队列**: [Asynq v0.25+](https://github.com/hibiken/asynq) - 基于 Redis 的轻量级异步任务/延迟任务队列
- **身份认证**: golang-jwt v5 - 双 Token（Access & Refresh）刷新与复用检测机制
- **实时通信**: gorilla/websocket - 结构化推送，支持 JWT 鉴权
- **热重载**: [air v1.51+](https://github.com/air-verse/air) - 实时监听代码变更自动重启服务

### 前端 (React)

- **核心框架**: React 19 + TypeScript

- **构建工具**: Vite 6.x - 极速的热重载与构建打包
- **UI 组件库**: Chakra UI v2 - 现代化、易用且高度可定制的样式系统
- **路由管理**: React Router v6
- **表格数据**: TanStack Table v8
- **多语言国际化**: i18next & react-i18next

---

## ✨ 核心功能

- 🔐 **安全认证**：JWT 双 Token 机制（Access + Refresh），支持 Refresh Token Rotation 刷新与重放/复用检测。密码使用 `bcrypt` (cost=12) 加密，敏感信息不记录日志。
- 👥 **RBAC 权限控制**：基于用户-角色-权限的多对多管理，中间件层实时拦截校验，并使用 Redis 进行高性能缓存。
- 📝 **审计日志**：自动记录所有非 GET 操作，通过 Asynq 队列异步落库，确保操作链路可追溯且防篡改。
- 📂 **文件上传管理**：内置文件分片上传、断点续传、秒传检测、多线程下载。支持 **本地存储 (Local)**、**数据库大对象 (PG lo)** 及 **S3 兼容存储 (如 MinIO/OSS)**。
- 🌐 **多语言国际化 (i18n)**：后端 API 错误消息和提示多语言统一翻译，前端界面字段和交互完整支持中英文切换。
- 💬 **WebSocket 实时推送**：集成实时通知推送功能，连接握手阶段安全集成 JWT 鉴权。
- ⚙️ **CRUD 代码生成器**：命令行一键根据 GORM Model 自动生成完整的 DTO、Handler、Service、Repository 和 Router 模板代码，节省大量重复劳动。
- 📄 **Swagger 接口文档**：注解驱动，支持配合代码生成器自动联动更新，开发环境下一键访问调试。

---

## 📐 系统架构

项目遵循典型的分层架构，单向依赖：

```txt
Handler (Controller) ──> Service (Business Logic) ──> Repository (Data Access) ──> Model (GORM)
         │                         │
         ▼                         ▼
   DTO (Validate)            Storage (local/pg/oss)
```

1. **Handler**: 负责参数绑定、基本格式校验（`go-playground/validator`）、调用 Service、组装统一格式响应。禁止编写业务逻辑。
2. **Service**: 承载核心业务逻辑，控制数据库事务，必要时进行细粒度权限校验。
3. **Repository**: 专注于底层 GORM 数据库操作，不掺杂任何业务逻辑。
4. **DTO**: 输入输出数据传输对象，使用 `validate` 标签声明校验规则。

---

## 🚀 快速开始

### 前置条件

确保本地已安装：

- Go 1.23+
- Node.js 18+ (推荐使用 pnpm)
- Docker & Docker Compose
- Make 工具

---

### 后端启动

1. **进入后端目录并一键初始化**：

   ```bash
   cd app
   make init
   ```

   *该命令将：复制 `.env.example` 为 `.env`、拉起 Docker 依赖容器（PostgreSQL & Redis）、拉取依赖包、执行数据库迁移初始化。*

2. **启动热重载开发服务**：

   ```bash
   make serve
   ```

   *服务启动后将监听 `http://localhost:8080`，并开启 Swagger 接口文档：`http://localhost:8080/swagger/index.html`。*

---

### 前端启动

1. **进入前端目录**：

   ```bash
   cd web
   ```

2. **安装依赖**：

   ```bash
   npm install  # 或使用 pnpm install / yarn
   ```

3. **启动开发服务**：

   ```bash
   npm run dev
   ```

   *前端开发服务默认启动在 `http://localhost:5173`。*

---

## 🐳 Docker 部署

项目包含完善的 Dockerfile 以及多环境的 docker-compose 配置，支持一键容器化部署。

### 1. 本地开发依赖环境 (PostgreSQL + Redis)

若只想在容器中跑数据库和缓存，而在本地运行 Go 和 React：

```bash
cd app
make docker-up-deps
```

*此操作等同于：`docker-compose up -d postgres redis`*

---

### 2. 本地一键容器化运行

如果想在本地将整个系统（包含 Go 后端、React 静态打包产物、DB、Redis）在容器中完整跑起来：

```bash
# 在项目根目录下执行
docker-compose up -d --build
```

- 构建时会通过多阶段构建（Multi-stage Build）自动在 node 容器中完成前端编译打包，并将 dist 产物直接复制给 Alpine 运行环境，由 Go 服务进行托管。

- 启动后一键访问：
  - 系统前端 & 接口服务：`http://localhost:8080`
  - 数据库监听端口：`5432`
  - Redis 监听端口：`6379`

---

### 3. 生产环境部署 (docker-compose.prod.yml)

生产环境下为保证安全，容器端口默认仅绑定在本地 `127.0.0.1`，建议前端使用 Nginx 反向代理接入，并配合环境变量注入密钥。

1. **复制并配置环境变量**：
   确保 `app/.env` 中包含您自定义的安全密码、JWT 秘钥等：

   ```bash
   NIKO_JWT_SECRET=your-production-secure-jwt-key
   NIKO_DB_PASSWORD=your-secure-db-password
   ```

2. **启动生产容器集群**：

   ```bash
   cd app
   make docker-prod
   ```

   *该命令将加载 `docker-compose.prod.yml` 构建并运行服务，日志、上传的文件均会通过外部 Docker Volume 进行持久化。*

3. **执行容器内数据库迁移**：

   ```bash
   make docker-migrate
   ```

4. **停止服务**：

   ```bash
   make docker-prod-down
   ```

---

## 🛠 日常开发常用命令

所有的后端开发命令均在 `app/` 目录下通过 `make` 执行：

```bash
make serve          # 启动本地依赖容器并运行 air 热重载服务
make dev            # 仅运行 air 热重载（适用于外部自行拉起的 DB & Redis）
make run            # 直接通过 go run 启动服务
make build          # 编译当前平台的二进制可执行文件
make build-linux    # 交叉编译 Linux amd64 平台二进制文件
make swag           # 重新生成 Swagger API 文档
make gen            # 运行代码生成器（为所有 Model 自动生成 CRUD 代码）
make unit-test      # 运行单元测试（包含 -race 与 -coverprofile 覆盖率报告）
make lint           # 使用 golangci-lint 进行代码静态检查
make migrate        # 运行数据库迁移脚本
make docker-up      # 后台启动所有容器服务 (含后端 Server + PostgreSQL + Redis)
make docker-down    # 停止并移除所有容器服务
make clean          # 清理编译产物和测试缓存
make help           # 查看所有的 make 指令及其说明
```

---

## 📁 项目目录结构

```
niko-admin/
├── app/                      # Go 后端项目根目录
│   ├── cmd/
│   │   ├── server/           # 主服务启动入口
│   │   ├── migrate/          # 数据库迁移脚本
│   │   └── gen/              # CRUD 代码生成 CLI 入口
│   ├── internal/
│   │   ├── config/           # 多环境配置加载 (Viper)
│   │   ├── middleware/       # Gin 中间件 (Auth、RBAC、CORS、i18n、Logger)
│   │   ├── router/           # API 路由分组注册
│   │   ├── handler/          # HTTP 控制器 (参数校验、数据响应)
│   │   ├── service/          # 业务逻辑与事务控制
│   │   ├── repository/       # 数据库数据访问 (GORM)
│   │   ├── model/            # 数据库实体定义 (GORM Models)
│   │   ├── dto/              # 输入输出数据契约 (Request/Response DTO)
│   │   ├── pkg/              # 内部基础公共组件 (JWT、Crypto、统一错误响应)
│   │   └── task/             # Asynq 异步任务注册与处理器
│   ├── pkg/
│   │   ├── gen/              # 代码生成器核心模板与引擎
│   │   └── storage/          # 文件存储模块抽象 (支持 local/pg/oss)
│   ├── configs/              # 各环境 YAML 静态配置文件
│   ├── docs/                 # 自动生成的 Swagger 接口文档目录
│   ├── Makefile              # 常用开发指令脚本
│   ├── Dockerfile            # 后端多阶段构建 Dockerfile
│   └── go.mod
├── web/                      # React 前端项目根目录
│   ├── src/                  # 前端源码
│   ├── public/               # 公共静态资源
│   ├── index.html            # 单页面入口 HTML
│   ├── vite.config.ts        # Vite 配置
│   └── package.json          # Node 依赖及脚本
├── docs/                     # 系统设计及产品文档
│   └── PRD.md
├── openspec/                 # 开放规格提案
├── docker-compose.yml        # 本地开发环境编排 (PostgreSQL, Redis, server)
└── docker-compose.prod.yml   # 生产环境编排配置
```

---

## 📝 API 规范与响应格式

### 统一 JSON 响应格式

```go
// 成功响应 (HTTP 200 OK)
response.OK(c, data)

// 错误响应 (HTTP Error Code)
response.Err(c, errors.New(code, message))

// 分页列表响应
response.Page(c, list, total, page, pageSize)
```

后端吐出的 JSON 格式如下：

- **操作成功**：

  ```json
  {
    "code": 0,
    "message": "success",
    "data": {
      "id": 1,
      "username": "admin"
    }
  }
  ```

- **操作失败** (带错误码与本地化翻译消息)：

  ```json
  {
    "code": 10001,
    "message": "用户名或密码错误"
  }
  ```

- **分页查询**：

  ```json
  {
    "code": 0,
    "message": "success",
    "data": {
      "list": [],
      "total": 100,
      "page": 1,
      "page_size": 20
    }
  }
  ```

---

## ⚙️ 代码生成器 (CRUD)

代码生成器能够扫描 `app/internal/model/` 目录下的 GORM Model，一键生成完整的 Handler、Service、Repository、Router 及 DTO 文件。

```bash
cd app

# 为单个 Model（例如 user）生成全套 CRUD 代码
go run cmd/gen/main.go user

# 为所有定义的 Model 一键生成全套 CRUD 代码
go run cmd/gen/main.go --all

# 查看所有支持生成代码的 Model 列表
go run cmd/gen/main.go --list
```

> 💡 **提示**：生成完毕后，通常只需在 `app/internal/router/router.go` 中注册对应的路由组即可完成新模块的上线。

---

## ⚙️ 配置文件说明

项目支持从 **环境变量、`.env` 文件、YAML 配置文件** 三种形式加载配置，优先级依次递增（环境变量最高）。

本地开发请复制 `app/.env.example` 并重命名为 `app/.env`。核心配置项说明：

```ini
NIKO_APP_ENV=dev                      # 运行环境 (dev / test / prod)
NIKO_APP_PORT=8080                     # 后端服务监听端口
NIKO_DB_HOST=localhost                 # PostgreSQL 主机地址
NIKO_DB_PORT=5432                      # PostgreSQL 端口
NIKO_DB_USER=postgres                  # PostgreSQL 用户名
NIKO_DB_PASSWORD=postgres              # PostgreSQL 密码
NIKO_DB_NAME=niko_admin                # PostgreSQL 数据库名
NIKO_REDIS_HOST=localhost              # Redis 主机地址
NIKO_REDIS_PORT=6379                   # Redis 端口
NIKO_JWT_SECRET=change-me-in-production# JWT 秘钥（生产环境必须修改）
NIKO_STORAGE_DRIVER=local              # 文件存储驱动 (local / pg / oss)
```

---

## 📚 相关文档

- [PRD 产品需求文档](./docs/PRD.md)
- Swagger 接口文档：`http://localhost:8080/swagger/index.html`（服务启动后访问）

---

## 📄 License

[MIT License](./LICENSE)
