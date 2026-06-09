# AIVisionInference

<p align="center">
  <strong>简体中文</strong> | <a href="./README_EN.md">English</a>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go Version"></a>
  <a href="https://gin-gonic.com/"><img src="https://img.shields.io/badge/Gin-v1.10-blue?logo=go" alt="Gin"></a>
  <a href="https://gorm.io/"><img src="https://img.shields.io/badge/GORM-v1.26-5c6bc0?logo=go" alt="GORM"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL"></a>
  <a href="https://redis.io/"><img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis"></a>
  <a href="https://isocpp.org/"><img src="https://img.shields.io/badge/C++-17-00599C?logo=cplusplus&logoColor=white" alt="C++17"></a>
  <a href="https://flatbuffers.dev/"><img src="https://img.shields.io/badge/FlatBuffers-IPC-orange" alt="FlatBuffers"></a>
  <a href="https://vite.dev/"><img src="https://img.shields.io/badge/Vite-6.x-646CFF?logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://chakra-ui.com/"><img src="https://img.shields.io/badge/Chakra--UI-2.x-319795?logo=chakra-ui&logoColor=white" alt="Chakra UI"></a>
</p>

AIVisionInference 是面向边缘设备与视频流场景的 **AI 视觉推理平台**。项目由 Go 控制面、React 管理端、C++ 推理数据面、ZLMediaKit 流媒体服务、算法包 ABI 与 FlatBuffers IPC 协议组成，支持设备接入、算法包管理、推理任务编排、智能记录沉淀与系统运维配置。

---

## 📖 目录

- [核心能力](#-核心能力)
- [系统架构](#-系统架构)
- [技术栈](#-技术栈)
- [快速开始](#-快速开始)
- [模块说明](#-模块说明)
- [常用命令](#-常用命令)
- [算法包规范](#-算法包规范)
- [IPC 协议](#-ipc-协议)
- [配置说明](#-配置说明)
- [相关文档](#-相关文档)

---

## ✨ 核心能力

- 🎥 **视频流与设备管理**：支持设备、通道、媒体流、GB28181/RTSP 相关资源管理，并通过 ZLMediaKit 承载流媒体能力。
- 🧠 **AI 推理任务编排**：Go 控制面下发任务，C++ 数据面执行流处理、算法加载、推理与结果上报。
- 📦 **算法包管理**：支持人脸识别、跌倒检测等算法包示例，算法以动态库和元数据形式接入。
- 🧾 **智能记录沉淀**：推理结果、告警事件、截图/证据与任务状态可回传控制面并落库。
- 🔐 **后台管理能力**：继承 Niko Admin 的 JWT 双 Token、RBAC、审计日志、文件管理、i18n 与 Swagger 能力。
- ⚙️ **边缘硬件适配**：C++ engine 支持 FFmpeg/OpenCV fallback，并预留 RKMPP/RGA 等硬件加速 HAL。

---

## 📐 系统架构

```txt
┌──────────────────────┐
│ React 管理端 web/     │
└──────────┬───────────┘
           │ HTTP / WebSocket
┌──────────▼───────────┐        FlatBuffers IPC        ┌──────────────────────┐
│ Go 控制面 app/        │ ───────────────────────────▶ │ C++ 推理引擎 engine/  │
│ Gin + GORM + Redis    │ ◀─────────────────────────── │ Pipeline + Algo .so   │
└──────┬─────────┬──────┘      状态 / 结果 / 指标        └──────────┬───────────┘
       │         │                                                  │
       ▼         ▼                                                  ▼
 PostgreSQL   Redis / Asynq                                  RTSP / HAL / NPU
       │
       ▼
 ZLMediaKit / 文件存储 / 智能记录
```

后端仍遵循分层架构：

```txt
Handler → Service → Repository → Model
   │         │
   ▼         ▼
  DTO     Storage / IPC / Task
```

---

## 🛠 技术栈

### 控制面与管理端

- **后端**：Go 1.23+、Gin、GORM、PostgreSQL 16、Redis 7、Asynq、JWT、Wire、Viper、Zap、Swagger。
- **前端**：React 19、TypeScript、Vite 6、Chakra UI 2、React Router、TanStack Table、i18next。

### 数据面与协议

- **推理引擎**：C++17、CMake、FlatBuffers、FFmpeg、OpenCV、动态库算法 ABI。
- **硬件适配**：RKMPP、RGA、DMA Buffer、VideoToolbox/macOS stub、x86_64 stub fallback。
- **流媒体**：ZLMediaKit，项目根目录 `docker-compose.yml` 默认使用 `zlmediakit/zlmediakit:latest`。

---

## 🚀 快速开始

### 前置条件

- Go 1.23+
- Node.js 18+ / npm
- Docker & Docker Compose
- CMake、C++17 编译器、FlatBuffers `flatc`
- 可选：FFmpeg、OpenCV、Rockchip MPP/RGA 开发包

### 1. 启动数据库与缓存

```bash
cd app
make docker-up-deps
```

该命令会在仓库根目录启动 `postgres` 与 `redis`。当前 `docker-compose.yml` 默认数据库配置为：

```txt
POSTGRES_USER=aivision
POSTGRES_PASSWORD=aivision123
POSTGRES_DB=aivision
```

### 2. 初始化并启动 Go 控制面

```bash
cd app
cp -n .env.example .env
make migrate
make serve
```

服务默认监听：

- API：`http://localhost:8080`
- WebSocket：`http://localhost:8090`
- Swagger：`http://localhost:8080/swagger/index.html`

### 3. 启动 React 管理端

```bash
cd web
npm install
npm run dev
```

前端开发服务默认监听 `http://localhost:5173`。

### 4. 构建 C++ 推理引擎

```bash
cd engine
make dev
```

开发构建默认关闭 RKMPP，使用 stub HAL，产物位于 `engine/build/`。

### 5. 一体化容器编排

```bash
docker-compose up -d --build
```

根目录 `docker-compose.yml` 编排了 PostgreSQL、Redis、ZLMediaKit、Go 控制面与 C++ engine。注意：当前仓库已包含 `deploy/Dockerfile.engine`，但 `go-server` 配置引用的 `deploy/Dockerfile` 未在当前目录中发现；如需完整容器化运行，请先补齐 Go 服务镜像构建文件或调整 compose 配置。

---

## 📁 模块说明

```txt
AIVisionInference/
├── app/                    # Go 控制面：API、RBAC、设备、算法、任务、记录、IPC、迁移
├── web/                    # React 管理端：设备、媒体预览、算法包、AI 任务、系统配置等页面
├── engine/                 # C++ 推理数据面：流处理、HAL、算法动态库加载、结果上报
├── algorithms/             # 算法包示例与模型资源，如 face_recognition、fall_detection
├── proto/flatbuf/          # Go/C++ IPC FlatBuffers schema 与兼容性说明
├── zlm/                    # ZLMediaKit 相关配置、数据与源码依赖
├── deploy/                 # 部署文件：engine Dockerfile、systemd、网络回滚脚本
├── docs/                   # 产品与设计文档
├── openspec/               # OpenSpec 变更提案与规格
└── docker-compose.yml      # 本地服务编排
```

---

## 🛠 常用命令

### Go 控制面（`app/`）

```bash
make serve          # 启动依赖容器并运行 air 热重载服务
make dev            # 仅运行 air 热重载，需自行启动 DB/Redis
make run            # go run 启动服务
make build          # 编译当前平台二进制
make build-linux    # 交叉编译 Linux amd64 二进制
make migrate        # 执行数据库迁移
make wire           # 重新生成 Wire 依赖注入代码
make swag           # 重新生成 Swagger 文档
make gen            # 基于 GORM Model 生成 CRUD 代码
make unit-test      # 运行 Go 单元测试
make lint           # 运行 golangci-lint
```

### React 管理端（`web/`）

```bash
npm run dev         # 启动 Vite 开发服务
npm run build       # TypeScript 检查并构建生产包
npm run preview     # 预览生产构建
npm run test        # 运行 Vitest
```

### C++ 推理引擎（`engine/`）

```bash
make dev            # x86_64/macOS 开发构建，stub HAL
make release        # Release 构建
make macos          # macOS 开发构建
make rk3568         # 在 RK3568 目标机本地构建
make rk3568-cross SYSROOT=/path/to/sysroot  # RK3568 交叉编译
make flatbuf        # 生成 C++ FlatBuffers 头文件
make test           # 编译并运行 engine 测试
make clean          # 清理构建目录
```

---

## 📦 算法包规范

算法包位于 `algorithms/`，通常包含：

```txt
algorithm-name/
└── version/
    ├── algo_meta.yaml      # 算法元数据、输入输出、阈值、运行参数
    ├── libxxx.so           # 算法动态库
    ├── models/             # 模型文件
    ├── label_map.json      # 标签映射
    └── examples/           # 示例图片或测试数据
```

C++ engine 通过动态库加载算法，算法包需遵循统一 C ABI，例如初始化、推理和销毁入口。具体可参考：

- `algorithms/face_recognition/1.0.0/README.md`
- `algorithms/fall_detection/1.0.0/README.md`

---

## 🔌 IPC 协议

Go 控制面与 C++ 数据面使用 `proto/flatbuf/` 下的 FlatBuffers schema 通信，核心消息包括：

- `StartStreamCmd` / `StopStreamCmd`：启动或停止视频流推理任务。
- `UpdateConfigCmd`：更新任务或算法参数。
- `LoadAlgoCmd` / `UnloadAlgoCmd`：加载或卸载算法包。
- `InferenceResultMsg`：上报推理结果。
- `StreamStatusMsg`：上报流状态。
- `EngineMetricsMsg`：上报引擎指标。
- `HeartbeatCmd` / `HeartbeatAckMsg`：双向心跳。

协议详情见 `proto/flatbuf/README.md`、`proto/flatbuf/CHANGELOG.md` 与 `proto/flatbuf/COMPATIBILITY.md`。

---

## ⚙️ 配置说明

配置加载优先级：`环境变量 > .env > config.yaml > 默认值`。所有环境变量统一使用 `NIKO_` 前缀，生产环境必须通过环境变量注入密钥。

常用配置示例：

```ini
NIKO_APP_ENV=dev
NIKO_APP_PORT=8080
NIKO_DB_HOST=localhost
NIKO_DB_PORT=5432
NIKO_DB_USER=aivision
NIKO_DB_PASSWORD=aivision123
NIKO_DB_NAME=aivision
NIKO_REDIS_HOST=localhost
NIKO_REDIS_PORT=6379
NIKO_JWT_SECRET=change-me-in-production
NIKO_ENGINE_ADDR=127.0.0.1:9500
NIKO_STORAGE_DRIVER=local
```

安全要求：

- 生产环境必须修改 `NIKO_JWT_SECRET`、数据库密码等敏感配置。
- 禁止将真实密钥提交到 Git。
- 上传文件、算法包与模型文件应按部署环境配置持久化路径。

---

## 📚 相关文档

- `engine/README.md`：C++ 推理引擎与 RKMPP/RGA 流水线说明。
- `proto/flatbuf/README.md`：FlatBuffers IPC 协议说明。
- `algorithms/face_recognition/1.0.0/README.md`：人脸识别算法包示例。
- `algorithms/fall_detection/1.0.0/README.md`：跌倒检测算法包示例。
- `prd/prd-draft.md`：产品需求草案。
- `prd/tech-design.md`：技术设计文档。
- `docs/gb28181-guide.md`：GB28181 接入说明。
- `docs/zlm-verification.md`：ZLMediaKit 验证说明。
- Swagger：服务启动后访问 `http://localhost:8080/swagger/index.html`。

---

## 📄 License

如需开源或分发，请根据项目实际授权策略补充 License 文件与版权说明。
