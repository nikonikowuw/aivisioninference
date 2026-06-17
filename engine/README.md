# aivision-engine — RKMPP/RGA 硬件加速流水线

## 架构

```
aivision-engine (主进程)
  │
  ├── libaivision-hal-rkmpp.so  ← 动态加载 (dlopen)
  │     └── RKMPPPipeline       ← Rockchip 硬件解码/编码/RGA
  │
  ├── libaivision-hal-macos-videotoolbox.so  (macOS)
  │     └── VideoToolboxPipeline
  │
  └── FLatBuffers IPC + 算法 .so

配置: hal_so_path 指向对应平台的 .so
```

## 构建

### 开发机 (x86_64) — Stub 模式

```bash
mkdir build && cd build
cmake .. -DAIVISION_WITH_RKMPP=OFF
make -j4
```

生成 `libaivision-hal-rkmpp.so` (Stub 模式)

### RK3568 目标机 — 本地编译（推荐）

```bash
# 安装依赖 (RK3588/RK356x Debian 系统)
sudo apt install rockchip-mpp-dev librga-dev

# 编译
mkdir build && cd build
cmake .. -DAIVISION_WITH_RKMPP=ON -DCMAKE_BUILD_TYPE=Release
make -j4
```

### 交叉编译（需要 SDK sysroot）

```bash
./scripts/build-rk3568.sh /path/to/rk3568/sysroot
```

## 配置

引擎启动时按 `默认值 < .env < 环境变量 < 命令行参数` 加载配置。默认读取当前工作目录 `.env`，也可以通过 `NIKO_ENGINE_ENV_FILE` 或 `--env-file` 指定文件。HAL 可以直接指定动态库路径，也可以指定平台名由引擎映射默认库路径。

### HAL 平台选择

| 平台                | 平台名/别名                                        | 默认 HAL 库                                |
| ------------------- | -------------------------------------------------- | ------------------------------------------ |
| Apple Silicon/macOS | `macos`, `mac`, `apple`, `mseries`, `videotoolbox` | `libaivision-hal-macos-videotoolbox.dylib` |
| Rockchip            | `rkmpp`, `rknn`, `rockchip`, `rk3568`, `rk3588`    | `libaivision-hal-rkmpp.so`                 |
| 华为昇腾            | `ascend`, `atlas`, `huawei`, `cann`                | `libaivision-hal-ascend.so`                |

默认 HAL 目录为 `/usr/local/lib/aivision`，可通过 `NIKO_ENGINE_HAL_DIR` 覆盖。

```bash
# Apple Silicon/macOS
NIKO_ENGINE_HAL_PLATFORM=macos aivision-engine

# Rockchip/RK3568/RK3588
NIKO_ENGINE_HAL_PLATFORM=rkmpp aivision-engine

# 华为昇腾（需要提供对应 HAL 实现）
NIKO_ENGINE_HAL_PLATFORM=ascend aivision-engine

# 显式指定动态库路径优先级最高
NIKO_ENGINE_HAL_SO=/opt/aivision/lib/libaivision-hal-rkmpp.so aivision-engine
```

### .env 示例

```ini
NIKO_ENGINE_WORKERS=4
NIKO_ENGINE_HAL_PLATFORM=macos
NIKO_ENGINE_HAL_DIR=/usr/local/lib/aivision
NIKO_ENGINE_HAL_CONFIG={"rga_enable":true,"rga_output_width":640,"rga_output_height":480}
NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK=true
NIKO_ENGINE_RTSP_PUSH=rtsp://localhost:10554
NIKO_ENGINE_ENABLE_MQTT=true
NIKO_ENGINE_MQTT_BROKER=tcp://localhost:1883
```

```bash
# 默认读取当前目录 .env
aivision-engine

# 指定 .env 文件
NIKO_ENGINE_ENV_FILE=/etc/aivision/engine.env aivision-engine
aivision-engine --env-file /etc/aivision/engine.env
```

### 环境变量

| 变量                                 | 说明                                          |
| ------------------------------------ | --------------------------------------------- |
| `NIKO_ENGINE_ENV_FILE`               | `.env` 文件路径                               |
| `NIKO_ENGINE_WORKERS`                | Worker 线程数                                 |
| `NIKO_ENGINE_HAL_PLATFORM`           | 主 HAL 平台名                                 |
| `NIKO_ENGINE_HAL_SO`                 | 主 HAL 动态库路径，优先于平台名               |
| `NIKO_ENGINE_FALLBACK_HAL_PLATFORM`  | 备用 HAL 平台名                               |
| `NIKO_ENGINE_FALLBACK_HAL_SO`        | 备用 HAL 动态库路径，优先于备用平台名         |
| `NIKO_ENGINE_HAL_DIR`                | 平台名映射时使用的 HAL 库目录                 |
| `NIKO_ENGINE_HAL_CONFIG`             | HAL 配置 JSON                                 |
| `NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK` | 是否允许 FFmpeg fallback，支持 `true/false`   |
| `NIKO_ENGINE_PLATFORM_URL`           | 平台管理端 URL（用于心跳上报）                |
| `NIKO_ENGINE_NODE_ID`                | 边缘节点 ID（平台注册后获取）                 |
| `NIKO_ENGINE_AUTH_TOKEN`             | 引擎认证 Token（平台创建节点后获取）          |
| `NIKO_ENGINE_RTSP_PUSH`              | RTSP 推流地址                                 |
| `NIKO_ENGINE_DEVICE_PLATFORM`        | 强制指定设备监控探测的平台类型                |
| `NIKO_ENGINE_DEVICE_STORAGE_PATH`    | 设备监控探测存储容量和利用率的挂载点路径      |
| `NIKO_ENGINE_DEVICE_ENABLE_COMMANDS` | 是否允许执行外部命令（如 `nvidia-smi` 等）    |
| `NIKO_ENGINE_DEVICE_COMMAND_TIMEOUT` | 外部命令执行的最大超时时长（毫秒，默认 1500） |

命令行也支持同名能力：

```bash
aivision-engine --hal-platform macos
aivision-engine --hal-platform rkmpp --fallback-hal-platform macos
aivision-engine --hal-so /opt/aivision/lib/libaivision-hal-rkmpp.so --hal-config '{"rga_enable":true}'
```

## RGA 配置项

| 字段                | 类型 | 默认值 | 说明              |
| ------------------- | ---- | ------ | ----------------- |
| `rga_enable`        | bool | false  | 启用 RGA 硬件缩放 |
| `rga_output_width`  | int  | -      | 缩放目标宽度      |
| `rga_output_height` | int  | -      | 缩放目标高度      |

## 编码器配置项 (EncodeInit)

| 字段      | 类型   | 默认值  | 说明              |
| --------- | ------ | ------- | ----------------- |
| `codec`   | string | h264    | h264 或 h265/hevc |
| `width`   | int    | 1920    | 编码宽度          |
| `height`  | int    | 1080    | 编码高度          |
| `bitrate` | int    | 4000000 | 目标码率 bps      |
| `fps`     | int    | 25      | 帧率              |
| `gop`     | int    | 50      | GOP 大小          |

## 边缘节点功能

引擎支持作为边缘节点连接到 Niko Admin 平台，实现状态上报、算法包下发和远程管理。

### Heartbeat Reporter

引擎每 5 秒向平台上报一次心跳，包含：

- 运行时长（uptime）
- 当前负载（current_load）
- 硬件信息（hardware_info）
- 已安装算法列表（installed_algorithms）
- 引擎版本（engine_version，当前版本 `1.0.0`）

心跳响应中包含待下发的算法包信息（pending_deployments），引擎自动下载并安装。

### 设备与指标监控 (Device Monitor)

引擎内置 `DeviceMonitor` 子系统，后台自动收集硬件规格、运行指标及 NPU/GPU 状态。

- **静态规格 (10 min)**：系统型号、主机名、OS、内存/存储总量。
- **轻量指标 (5 sec)**：CPU/内存利用率、存储利用率、温度。
- **外部探测 (20 sec)**：执行 `nvidia-smi` 或 `npu-smi`，支持失败退避逻辑。
- **集成**：支持 HTTP API (`/api/engine/device`) 和心跳包异步同步。

### 算法包下载与安装

1. 平台管理员触发算法包下发
2. 平台在心跳响应中返回下载 URL（预签名 URL，1 小时有效期）
3. 引擎下载 tar.gz 文件到 /tmp
4. 引擎校验 MD5
5. 引擎解压并加载 .so 算法文件
6. 下次心跳时上报安装状态

### 版本兼容性

引擎在心跳中携带版本号（`AIVISION_ENGINE_VERSION`），当前版本 `1.0.0`。
平台可配置最低兼容版本，过旧引擎的心跳将被拒绝。

### 配置指南

完整配置示例见 [.env.example](./.env.example)。

```ini
# 平台连接
NIKO_ENGINE_PLATFORM_URL=http://your-platform:8080
NIKO_ENGINE_NODE_ID=<从平台获取>
NIKO_ENGINE_AUTH_TOKEN=<从平台获取>

# HAL 配置
NIKO_ENGINE_HAL_PLATFORM=macos
NIKO_ENGINE_HAL_DIR=/usr/local/lib/aivision
```

## 数据流

```
RTSP TCP ─→ RTP (交织模式 0x24)
   │
   ├── H.264: FU-A/STAP-A 解包
   └── H.265: FU/AP 解包
              │
              ▼
         MPP Decoder ─→ HwBuffer (DMA fd, 零拷贝)
              │
         ┌────┴────┐
         ▼         ▼
    RGA Resize  HwBuffer → 推理
         │
         ▼
    HwBuffer → MPP Encoder → H.264/H.265 码流
```

## 测试

```bash
cmake .. -DAIVISION_WITH_RKMPP=OFF
make test_rkmpp_pipeline
./test_rkmpp_pipeline
```
