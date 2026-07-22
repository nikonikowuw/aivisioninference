# aivision-engine — 硬件加速推理引擎

## 架构

HAL (Hardware Abstraction Layer) 各平台实现编译时直接链接进引擎，通过 `#ifdef` 选择。不再使用 dlopen 插件机制。

```
aivision-engine (主进程，含 HAL 静态链接)
  │
  ├── macOS:    VideoToolboxPipeline   (VideoToolbox 硬解/硬编)
  ├── Rockchip: RKMPPPipeline          (MPP 硬解/硬编 + RGA 缩放)
  │
  ├── FFmpeg fallback (软解，兜底)
  └── FlatBuffers payloads + 算法 .so
```

## 构建

### macOS

```bash
mkdir build && cd build
cmake .. -DCMAKE_INSTALL_PREFIX=/usr/local
make -j$(sysctl -n hw.ncpu)
```

### RK3568 目标机 — 本地编译

```bash
sudo apt install rockchip-mpp-dev librga-dev
mkdir build && cd build
cmake .. -DAIVISION_WITH_RKMPP=ON -DCMAKE_BUILD_TYPE=Release
make -j4
```

### 交叉编译（需要 SDK sysroot）

```bash
./scripts/build-rk3568.sh /path/to/rk3568/sysroot
```

## 配置

引擎启动时按 `默认值 < .env < 环境变量 < 命令行参数` 加载配置。默认读取当前工作目录 `.env`，也可以通过 `NIKO_ENGINE_ENV_FILE` 或 `--env-file` 指定文件。

### .env 示例

```ini
NIKO_ENGINE_WORKERS=4
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
| `NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK` | 是否允许 FFmpeg fallback，支持 `true/false`   |
| `NIKO_ENGINE_PLATFORM_URL`           | 平台管理端 URL（用于心跳上报）                |
| `NIKO_ENGINE_NODE_ID`                | 边缘节点 ID（平台注册后获取）                 |
| `NIKO_ENGINE_AUTH_TOKEN`             | 引擎认证 Token（平台创建节点后获取）          |
| `NIKO_ENGINE_RTSP_PUSH`              | RTSP 推流地址                                 |
| `NIKO_ENGINE_DEVICE_PLATFORM`        | 强制指定设备监控探测的平台类型                |
| `NIKO_ENGINE_DEVICE_STORAGE_PATH`    | 设备监控探测存储容量和利用率的挂载点路径      |
| `NIKO_ENGINE_DEVICE_ENABLE_COMMANDS` | 是否允许执行外部命令（如 `nvidia-smi` 等）    |
| `NIKO_ENGINE_DEVICE_COMMAND_TIMEOUT` | 外部命令执行的最大超时时长（毫秒，默认 1500） |

## 数据流

```
RTSP TCP ─→ RTP (交织模式 0x24)
   │
   ├── H.264: FU-A/STAP-A 解包
   └── H.265: FU/AP 解包
              │
              ▼
   硬件解码器 ─→ HwBuffer (零拷贝)
         │
    ┌────┴────┐
    ▼         ▼
  硬件缩放   HwBuffer → 推理 Worker
    │
    ▼
  硬件编码器 → H.264/H.265 码流
```

## 测试

```bash
cmake .. -DAIVISION_WITH_RKMPP=OFF
make test_rkmpp_pipeline
./test_rkmpp_pipeline
```
