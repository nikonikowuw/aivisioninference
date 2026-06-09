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

引擎配置 JSON 中指定 `hal_so_path`:

```json
{
    "device_id": "camera01",
    "hal_so_path": "/usr/lib/aivision/libaivision-hal-rkmpp.so",
    "hal_config": {
        "rga_enable": true,
        "rga_output_width": 640,
        "rga_output_height": 480
    }
}
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
