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

生成 `libaivision-hal-rkmpp.so`（不包含 MPP/RGA 功能，仅用于测试接口）

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

| 平台 | 平台名/别名 | 默认 HAL 库 |
|------|-------------|-------------|
| Apple Silicon/macOS | `macos`, `mac`, `apple`, `mseries`, `videotoolbox` | `libaivision-hal-macos-videotoolbox.dylib` |
| Rockchip | `rkmpp`, `rknn`, `rockchip`, `rk3568`, `rk3588` | `libaivision-hal-rkmpp.so` |
| 华为昇腾 | `ascend`, `atlas`, `huawei`, `cann` | `libaivision-hal-ascend.so` |

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
NIKO_ENGINE_IPC_ADDR=0.0.0.0:9500
NIKO_ENGINE_WORKERS=4
NIKO_ENGINE_HAL_PLATFORM=macos
NIKO_ENGINE_HAL_DIR=/usr/local/lib/aivision
NIKO_ENGINE_HAL_CONFIG={"rga_enable":true,"rga_output_width":640,"rga_output_height":480}
NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK=true
NIKO_ENGINE_RTSP_PUSH=rtsp://localhost:10554
```

```bash
# 默认读取当前目录 .env
aivision-engine

# 指定 .env 文件
NIKO_ENGINE_ENV_FILE=/etc/aivision/engine.env aivision-engine
aivision-engine --env-file /etc/aivision/engine.env
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `NIKO_ENGINE_ENV_FILE` | `.env` 文件路径 |
| `NIKO_ENGINE_IPC_ADDR` / `NIKO_ENGINE_ADDR` | IPC 监听地址 |
| `NIKO_ENGINE_WORKERS` | Worker 线程数 |
| `NIKO_ENGINE_HAL_PLATFORM` | 主 HAL 平台名 |
| `NIKO_ENGINE_HAL_SO` | 主 HAL 动态库路径，优先于平台名 |
| `NIKO_ENGINE_FALLBACK_HAL_PLATFORM` | 备用 HAL 平台名 |
| `NIKO_ENGINE_FALLBACK_HAL_SO` | 备用 HAL 动态库路径，优先于备用平台名 |
| `NIKO_ENGINE_HAL_DIR` | 平台名映射时使用的 HAL 库目录 |
| `NIKO_ENGINE_HAL_CONFIG` | HAL 配置 JSON |
| `NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK` | 是否允许 FFmpeg fallback，支持 `true/false` |

命令行也支持同名能力：

```bash
aivision-engine --hal-platform macos
aivision-engine --hal-platform rkmpp --fallback-hal-platform macos
aivision-engine --hal-so /opt/aivision/lib/libaivision-hal-rkmpp.so --hal-config '{"rga_enable":true}'
```

## RGA 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `rga_enable` | bool | false | 启用 RGA 硬件缩放 |
| `rga_output_width` | int | - | 缩放目标宽度 |
| `rga_output_height` | int | - | 缩放目标高度 |

## 编码器配置项 (EncodeInit)

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `codec` | string | h264 | h264 或 h265/hevc |
| `width` | int | 1920 | 编码宽度 |
| `height` | int | 1080 | 编码高度 |
| `bitrate` | int | 4000000 | 目标码率 bps |
| `fps` | int | 25 | 帧率 |
| `gop` | int | 50 | GOP 大小 |

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
