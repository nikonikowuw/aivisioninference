# 边缘节点平台及指标探测兼容性矩阵

本文档说明 AIVisionInference 边缘推理引擎在不同硬件/操作系统平台下的指标探测能力及已知限制。

## 平台兼容性概览

| 平台标识 (Platform ID) | 操作系统 | 支持的硬件加速器/处理器类型 | 指标数据源 | 采集机制 | 限制与降级说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`linux_generic`** | Linux (Generic) | CPU (x86_64, AArch64 等) | `/proc/stat`, `/proc/meminfo`, `statvfs` | 轻量级文件读取, `statvfs` 系统调用 | 仅包含系统基础指标，无硬件加速器（GPU/NPU）专属指标 |
| **`rockchip`** | Linux | Rockchip RK3588 / RK3568 SoC | `/sys/kernel/debug/rknpu/load`, `/sys/class/thermal/` | Debugfs/Sysfs 节点读取 | NPU 核心指标独立上报，总使用率通过多核求和（Sum）并 Clamp 至 100% 统计 |
| **`nvidia`** | Linux | NVIDIA Tesla / GeForce, NVIDIA Jetson | `nvidia-smi` 命令行, `/sys/devices/gpu.0/load` | 周期性外部命令执行 / Jetson Sysfs 读取 | 外部命令执行带超时（默认 1.5s）和指数退避退避保护；多 GPU 时系统汇总使用率取最大值（Max） |
| **`ascend`** | Linux | 华为昇腾 (Ascend 310/910 等) | `npu-smi` 命令行 | 周期性外部命令执行（支持表格和键值对格式） | 外部命令执行带超时和退避保护；多 NPU 时系统汇总使用率取最大值（Max） |
| **`macos`** | macOS | Apple Silicon (M1/M2/M3 等) | `sysctl`, Mach VM stats, `host_statistics` | macOS 专有 Mach API 和 sysctl 调用 | Apple GPU 和 Neural Engine 仅上报静态型号，动态使用率标记为 `unavailable` |

---

## 详细功能矩阵

### 1. 系统基础指标 (System Metrics)

| 指标字段 | `linux_generic` | `rockchip` | `nvidia` | `ascend` | `macos` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **主机名 (Hostname)** | `gethostname()` | `gethostname()` | `gethostname()` | `gethostname()` | `gethostname()` |
| **操作系统 (OS/Kernel)** | `uname()` | `uname()` | `uname()` | `uname()` | `uname()` |
| **CPU 型号 (CPU Model)** | `/proc/cpuinfo` | `/proc/cpuinfo` | `/proc/cpuinfo` | `/proc/cpuinfo` | `sysctl("machdep.cpu.brand_string")` |
| **CPU 核心数 (CPU Cores)** | `sysconf` | `sysconf` | `sysconf` | `sysconf` | `sysctl("hw.ncpu")` |
| **物理内存总量 (Total Mem)**| `/proc/meminfo` | `/proc/meminfo` | `/proc/meminfo` | `/proc/meminfo` | `sysctl("hw.memsize")` |
| **磁盘存储总量 (Total Disk)**| `statvfs` | `statvfs` | `statvfs` | `statvfs` | `statvfs` |
| **CPU 使用率 (CPU Usage)** | 双采样 Delta | 双采样 Delta | 双采样 Delta | 双采样 Delta | `HOST_CPU_LOAD_INFO` 双采样 Delta |
| **内存使用率 (Mem Usage)** | `MemAvailable` 比例| `MemAvailable` 比例| `MemAvailable` 比例| `MemAvailable` 比例| Mach VM pages (Active+Wire+Spec) |
| **磁盘使用率 (Disk Usage)**| `statvfs` f_bavail | `statvfs` f_bavail | `statvfs` f_bavail | `statvfs` f_bavail | `statvfs` f_bavail |
| **系统温度 (Temperature)** | N/A | Thermal zone scan | N/A | N/A | N/A |

### 2. 硬件加速器指标 (Accelerator Metrics)

| 指标字段 | `linux_generic` | `rockchip` | `nvidia` | `ascend` | `macos` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **加速器类型 (Type)** | N/A | `"npu"` | `"gpu"` | `"npu"` | `"gpu"` / `"npu"` |
| **厂商名称 (Vendor)** | N/A | `"rockchip"` | `"nvidia"` | `"ascend"` | `"apple"` |
| **设备型号 (Model)** | N/A | RKNPU v1/v2 (推导) | `nvidia-smi` / Tegra | `npu-smi info` | Apple GPU / Apple Neural Engine |
| **核心温度 (Temp)** | N/A | Sysfs (SoC/NPU) | `nvidia-smi` | `npu-smi info` | N/A (降级) |
| **利用率 (Usage)** | N/A | Multi-core sum | `nvidia-smi` / Sysfs | `npu-smi info` | N/A (标记为 `unavailable`) |
| **显存/内存使用率 (Mem)** | N/A | N/A | `nvidia-smi` | `npu-smi info` | N/A (标记为 `unavailable`) |
| **子核心分解 (Cores)** | N/A | Core0/1/2 独立使用率 | N/A | N/A | N/A |

---

## 关键机制与已知限制

### 1. 外部命令执行防挂死及指数退避保护
针对 `nvidia-smi` 和 `npu-smi` 等由于底层驱动挂起可能导致执行挂起的情况，推理引擎采用了以下安全设计：
- **执行超时限制**：所有外部命令行执行均由专有的 POSIX fork/exec 辅助类（`CommandRunner`）管理，并强制应用超时（默认 `1500ms`）。超时后，主进程将向子进程发送 `SIGKILL` 强行终止，以防止阻塞监控流水线。
- **指数退避重试 (Exponential Backoff)**：若某外部命令行执行失败或超时，该 Probe 将进入冷却状态，在冷却时间结束前不会重复发起调用，避免持续耗费系统资源或加剧驱动故障。冷却间隔按失败次数递增：`30秒`、`60秒`、`300秒`、`600秒`。任何一次成功执行都会立即将失败计数清零。

### 2. 多卡/多核指标聚合规则
- **多核心 NPU (如 RK3588)**：`rknpu/load` 提供多核心独立的利用率（如 `Core0: 10%, Core1: 20%, Core2: 5%`）。引擎在 `accelerators[0].cores` 数组中完整保留每核心的原始利用率，而在系统总体的 `metrics.npu_usage` 中上报其**算术累加值**（即 `35%`）。若累加值超过 `100%`，则截断在 `100%`。
- **多 GPU/NPU 卡 (如 NVIDIA Server, Ascend Server)**：若系统存在多个加速器卡，引擎将在 `accelerators` 数组中上报每个卡独立的利用率、显存和温度信息。在系统总体的汇总指标（`metrics.gpu_usage` 或 `metrics.npu_usage`）中，取所有卡中的**最大值 (Max)**（即热点核心利用率），以更灵敏地反映局部计算压力。

### 3. macOS 限制
在 macOS (Darwin) 系统上，由于 Apple Silicon 统一内存架构和专有硬件加速接口（Metal/ANE）的系统级 API 封闭性：
- 引擎不会在后台周期性拉取 Apple GPU/ANE 的实时利用率和显存，避免引起不必要的系统功耗。
- `accelerators` 中仍然会声明加速器的存在（`Apple GPU` 和 `Apple Neural Engine`），但其 `usage` 字段的 `available` 标记为 `false`，错误原因为 `"Not supported on macOS"`。
