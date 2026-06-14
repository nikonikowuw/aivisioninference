# 引擎配置与故障排查

本文档面向运维和开发人员，说明 C++ 推理引擎的边缘节点模式配置方法、依赖项和故障排查手段。

---

## 配置概览

引擎在边缘节点模式下需要以下配置才能接入平台：

### 必须配置

| 环境变量                   | 说明                                   | 获取方式                           |
|----------------------------|----------------------------------------|------------------------------------|
| `NIKO_ENGINE_PLATFORM_URL` | 平台管理端 URL (如 `http://10.0.0.1:8080`) | 根据实际部署地址填写               |
| `NIKO_ENGINE_NODE_ID`      | 边缘节点 ID                            | 平台创建节点后获取                 |
| `NIKO_ENGINE_AUTH_TOKEN`   | 引擎 JWT Token                         | 平台创建节点后获取（仅显示一次）   |

### 可选配置

| 环境变量                    | 默认值             | 说明                                       |
|-----------------------------|--------------------|--------------------------------------------|
| `NIKO_ENGINE_HTTP_PORT`     | `8080`             | 引擎本地 HTTP 服务端口                      |
| `NIKO_ENGINE_MINIO_ENDPOINT`| `""`               | MinIO 对象存储地址（算法包下载需要）        |
| `NIKO_ENGINE_IPC_ADDR`      | `0.0.0.0:9500`     | IPC 通信监听地址                            |
| `NIKO_ENGINE_WORKERS`       | `4`                | Worker 线程数                               |
| `NIKO_ENGINE_HAL_PLATFORM`  | `""`               | HAL 平台名 (`macos` / `rkmpp` / `ascend`)   |

### 完整 .env 示例

```ini
# ====================
# 平台连接（必须）
# ====================
NIKO_ENGINE_PLATFORM_URL=http://your-platform:8080
NIKO_ENGINE_NODE_ID=
NIKO_ENGINE_AUTH_TOKEN=

# ====================
# 引擎服务（可选）
# ====================
NIKO_ENGINE_HTTP_PORT=8080
NIKO_ENGINE_IPC_ADDR=0.0.0.0:9500
NIKO_ENGINE_WORKERS=4

# ====================
# 算法包存储（可选）
# ====================
NIKO_ENGINE_MINIO_ENDPOINT=http://minio:9000

# ====================
# HAL 配置（按需）
# ====================
NIKO_ENGINE_HAL_PLATFORM=macos
NIKO_ENGINE_HAL_DIR=/usr/local/lib/aivision
NIKO_ENGINE_HAL_CONFIG={}
NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK=true
NIKO_ENGINE_RTSP_PUSH=rtsp://localhost:10554

# ====================
# ZLM 配置（按需）
# ====================
NIKO_ENGINE_ZLM_URL=http://localhost:80
NIKO_ENGINE_ZLM_SECRET=
```

---

## 配置加载优先级

```
默认值 < .env 文件 < 环境变量 < 命令行参数
```

引擎启动时依次尝试加载配置：

1. 使用代码中的默认值
2. 加载当前目录 `.env` 文件（可通过 `NIKO_ENGINE_ENV_FILE` 或 `--env-file` 指定路径）
3. 读取同名的系统环境变量（覆盖 `.env` 中的值）
4. 解析命令行参数（最高优先级）

```bash
# 启动方式 1：使用当前目录 .env
./build/aivision-engine

# 启动方式 2：指定配置文件路径
NIKO_ENGINE_ENV_FILE=/etc/aivision/engine.env ./build/aivision-engine

# 启动方式 3：使用命令行参数
./build/aivision-engine --env-file /etc/aivision/engine.env

# 启动方式 4：环境变量直接传递
NIKO_ENGINE_PLATFORM_URL=http://platform:8080 ./build/aivision-engine
```

---

## 编译引擎

### macOS 开发机

```bash
cd engine
make dev
```

等价于：

```bash
mkdir -p build && cd build
cmake .. -DAIVISION_WITH_RKMPP=OFF -DCMAKE_BUILD_TYPE=Debug
make -j$(sysctl -n hw.ncpu)
```

产物位于 `engine/build/aivision-engine`。

### RK3568 目标机

```bash
cd engine
make rk3568
```

或手动：

```bash
mkdir -p build-rk3568 && cd build-rk3568
cmake .. -DAIVISION_WITH_RKMPP=ON -DCMAKE_BUILD_TYPE=Release
make -j4
```

### 交叉编译

需要 SDK sysroot：

```bash
./scripts/build-rk3568.sh /path/to/rk3568/sysroot
```

---

## 运行测试

```bash
cd engine/build
ctest --output-on-failure
```

测试覆盖：

| 测试项                    | 覆盖内容                         |
|---------------------------|----------------------------------|
| `test_http_server`        | HTTP Server 路由注册、健康检查   |
| `test_heartbeat_reporter` | HeartbeatReporter 心跳 JSON 构建 |
| `test_algorithm_downloader` | MD5 校验、tar.gz 提取逻辑     |
| `test_rkmpp_pipeline`     | RKMPP Pipeline 接口              |

---

## 内置 HTTP 接口

引擎启动后在 `NIKO_ENGINE_HTTP_PORT`（默认 8080）监听以下接口：

| 接口             | 方法 | 说明                         |
|------------------|------|------------------------------|
| `/health`        | GET  | 健康检查                     |
| `/hardware-info` | GET  | 硬件信息查询                 |
| `/deploy-algo`   | POST | 接收算法包部署请求           |
| `/algorithms`    | GET  | 查询已加载算法列表           |

健康检查示例：

```bash
curl http://localhost:8080/health
{"status":"ok","uptime":120,"current_load":1,"engine_version":"1.0.0"}
```

---

## 心跳机制

引擎通过 `HeartbeatReporter` 按以下流程上报状态：

```
[引擎]                          [平台]
  │                               │
  │──── POST /edge-nodes/:id/heartbeat ────▶
  │     { uptime, current_load,     │
  │       hardware_info,           │
  │       installed_algorithms,    │
  │       engine_version }         │
  │                               │
  │◀──── 200 OK ─────────────────┤
  │     { pending_deployments: [  │
  │         { download_url,       │
  │           expected_md5,       │
  │           extract_path } ] }  │
  │                               │
  │  ↓ 如果有待下发算法包           │
  │──── AlgorithmDownloader ─────▶│
  │     下载 → 校验 → 解压 → 加载  │
```

- **间隔**：每 5 秒
- **超时判定**：平台 15 秒未收到心跳标记为 offline
- **重连**：引擎自动重试（日志记录错误）

---

## 算法包下发流程

```
管理员触发下发
      │
      ▼
平台创建 EdgeNodeAlgorithm 记录（status: pending）
      │
      ▼
引擎心跳上报 → 平台在响应中返回 pending_deployments
      │
      ▼
引擎 AlgorithmDownloader 异步执行：
  1. libcurl 下载 tar.gz → /tmp/
  2. OpenSSL MD5 校验
  3. tar -xzf 解压到安装目录
  4. 搜索 .so 文件
  5. AlgoManager::Load() 加载动态库
      │
      ▼
下次心跳上报已安装状态（status: installed / failed）
```

---

## 故障排查

### 引擎无法启动

**检查项：**

1. 配置文件格式是否正确（无 BOM、无多余空格）
2. 端口是否被占用（默认 8080 / 9500）
3. `cmake ..` 阶段是否缺失依赖

```bash
# 检查端口占用
lsof -i :8080
lsof -i :9500
```

### 心跳上报失败

**检查日志**：引擎启动后查看 `[HeartbeatReporter]` 开头的输出。

常见错误：

| 错误信息                              | 可能原因                          |
|---------------------------------------|-----------------------------------|
| `Failed to resolve hostname`          | `NIKO_ENGINE_PLATFORM_URL` 不可达  |
| `HTTP 401` / `HTTP 403`               | `NIKO_ENGINE_AUTH_TOKEN` 无效或过期 |
| `HTTP 404`                            | `NIKO_ENGINE_NODE_ID` 不存在       |
| `Connection refused`                  | 平台服务未启动或端口不匹配         |
| `Timeout`                             | 网络不通或防火墙拦截               |

### 算法下载失败

**检查项：**

1. `NIKO_ENGINE_MINIO_ENDPOINT` 配置是否正确
2. 预签名 URL 是否过期（有效期 1 小时，过期后引擎下次心跳会重新获取）
3. 引擎所在设备能否访问 MinIO
4. 算法包文件是否存在于 MinIO Bucket

### 版本不兼容

- 平台默认最低兼容版本为 `"1.0.0"`
- 引擎版本通过 `InferenceEngine::Version()` 返回 `"1.0.0"`
- 可在 `app/configs/config.yaml` 中调整 `engine.min_compatible_version`
- 可通过 `engine.version_check_enabled: false` 临时关闭版本校验

---

## 平台侧部署配置要求

为确保边缘节点管理功能正常运转，平台控制面（Go 服务）也需要正确配置以下环境变量：

### 1. MinIO 存储访问配置

平台需要使用 MinIO 存储来存放下发的算法包，并为其生成预签名下载 URL。必须在平台环境（如 `.env` 文件或容器环境变量）中配置以下项：

| 环境变量 | 示例值 | 说明 |
|----------|--------|------|
| `NIKO_STORAGE_DRIVER` | `oss` | 启用对象存储作为存储驱动 |
| `NIKO_STORAGE_OSS_ENDPOINT` | `127.0.0.1:9000` | MinIO 服务的访问地址 |
| `NIKO_STORAGE_OSS_ACCESS_KEY` | `minioadmin` | MinIO 访问密钥 (Access Key) |
| `NIKO_STORAGE_OSS_SECRET_KEY` | `minioadmin` | MinIO 秘密密钥 (Secret Key) |
| `NIKO_STORAGE_OSS_BUCKET` | `aivision-algorithms` | 用于存储算法包的 Bucket 名称 |
| `NIKO_STORAGE_OSS_USE_SSL` | `false` | 是否启用 SSL 传输加密 |

> **安全提示**：请确保该 Bucket 策略设置为 `private`（禁止匿名访问），平台会自动通过预签名安全机制下发临时下载 URL 给引擎。

### 2. 引擎版本兼容性校验配置

平台支持对引擎上报的心跳进行版本强校验，防止因版本落后而导致推理任务执行异常。可在平台配置文件（如 `config.dev.yaml` / `config.prod.yaml`）或环境变量中配置：

| 环境变量/配置项 | 默认值 | 说明 |
|-----------------|--------|------|
| `NIKO_ENGINE_MIN_COMPATIBLE_VERSION` / `engine.min_compatible_version` | `"1.0.0"` | 引擎最低兼容版本号（遵循 SemVer 规范） |
| `NIKO_ENGINE_VERSION_CHECK_ENABLED` / `engine.version_check_enabled` | `true` | 是否启用版本兼容性校验。设为 `false` 可临时关闭校验 |
