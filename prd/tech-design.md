# AIVisionInference 边缘视觉推理系统 - 技术方案文档

## 1. 文档概述

本文档是 AIVisionInference 边缘视觉推理系统的技术架构与详细设计方案。本方案基于 PRD 需求，旨在构建一套高性能、可扩展、可运维的边缘视觉智能中枢系统，解决传统架构在多路并发、硬件适配、算法交付及运维闭环方面的痛点。

## 2. 总体架构设计

系统采用**控制面（Control Plane）与数据面（Data Plane）分离**的核心架构理念，保证业务编排的灵活性与底层推理的极致性能。

```mermaid
graph TD
    subgraph 前端与外部系统
        AdminWeb[Admin Web 前端]
        BizSys[第三方业务系统]
        Webhook[Webhook 告警接收]
    end

    subgraph Go 控制面 (Management Plane)
        API[REST API & Auth]
        DeviceMgr[设备与流管理器]
        AlgoMgr[算法包管理器]
        TaskSched[任务调度器]
        RuleEngine[规则引擎与事件路由]
        SIPServer[GB28181 SIP Server]
    end

    subgraph 媒体服务面 (Media Service)
        ZLM[ZLMediaKit]
    end

    subgraph 数据面 (分布式 C++ 推理节点 1..N)
        CPPEngine[C++ Inference Engine]
        TCP[Raw TCP Socket IPC]
        WorkerPool[Stream Worker Pool]
        Decoder[HW Decoder Adapter]
        Runtime[NPU Runtime Adapter]
        AlgoLoader[Algorithm Loader]
    end

    subgraph 基础设施 (Infrastructure)
        PG[(PostgreSQL + pgvector)]
        Redis[(Redis + Asynq)]
        Storage[Local/OSS Storage]
        Hardware[NPU: RKNN/CANN]
    end

    AdminWeb <--> API
    BizSys <--> API
    RuleEngine --> Webhook

    API <--> DeviceMgr
    API <--> AlgoMgr
    API <--> TaskSched

    DeviceMgr <--> SIPServer
    SIPServer <--> ZLM
    DeviceMgr --> ZLM : HTTP API / WebHook

    TaskSched <--> TCP : FlatBuffers 信令
    TCP <--> CPPEngine
    RuleEngine <--> TCP : FlatBuffers 结果接收

    ZLM --> WorkerPool : RTSP/RTP 流
    WorkerPool --> Decoder
    Decoder --> Runtime
    Runtime <--> AlgoLoader

    Go控制面 --> PG
    Go控制面 --> Redis
    Go控制面 --> Storage
```

### 2.1 架构分层说明

1. **Go 管理端（控制面）**：
   - 负责 API 暴露、RBAC 鉴权、业务编排。
   - 管理设备接入（集成 GB28181 SIP Server 接收注册和心跳）。
   - 管理算法包版本和授权。
   - 包含规则引擎，对 C++ 上报的结果进行过滤、去重并分发至 Webhook。
   - 异步处理图片特征提取、存储清理和 Webhook 重试。

2. **ZLMediaKit（媒体服务）**：
   - 统一屏蔽视频流协议差异。接收外部 RTSP 直连或 GB28181 点播推送的 RTP 流。
   - 输出前端可播放的 WebRTC/HTTP-FLV。
   - 为 C++ 引擎提供稳定可访问的本地流拉取地址。

3. **C++ 推理引擎（数据面）**：
   - 专注极致性能，单流单 Worker 或线程池管理。
   - 封装厂商底层硬解接口（Rockchip MPP / Huawei DVPP）。
   - 通过 NPU Runtime（RKNN / CANN）执行推理。
   - 通过 `dlopen` 动态加载标准算法 `.so` 库。

## 3. 核心机制设计

### 3.1 IPC 通信与零拷贝机制

Go 与 C++ 作为独立进程运行，以保证 C++ 算法崩溃时不会导致整个管理节点宕机。

- **控制信令与结果回传**：采用 **Raw TCP Socket + Length-Prefixed FlatBuffers** 进行双向通信。控制指令（启动、停止）、心跳和算法推理结果全部采用 FlatBuffers 序列化，支持跨机房或跨局域网调度多台 C++ 边缘节点。
- **单图零拷贝 (Zero-Copy for Images)**：
  - Go 端通过 `memfd_create` 创建匿名共享内存并写入图片数据。
  - 采用极简设计，取消 UDS 的文件描述符传递，单图推理直接将图像的 Base64 编码封装入 TCP 载荷。
  - C++ 端 `mmap` 该 FD 直接读取内存进行推理，消除了进程间的内存拷贝开销。
  - C++ 侧采用 RAII 模式严格管理 FD 和 mmap 内存的释放，防止 OOM。

### 3.2 算法包标准化与热更新

算法包是平台开放生态的核心，采用标准化压缩包交付。

- **结构规范**：
  必须包含 `algo_meta.yaml`（元数据）、`nikoniko_detector.so`（入口动态库）、`testimage.jpg`（自检图）、`label_map.json`（类别映射）。
- **C ABI 隔离**：
  C++ 不直接暴露类给主程序，而是导出 `create_detector`、`detector_infer`、`detector_free_result`、`destroy_detector` 四个纯 C 函数。主程序通过 `dlopen` 动态加载，避免了 C++ ABI 兼容性问题。
- **极致单进程多线程与自检机制**：
  废弃了复杂的跨进程沙箱隔离，改为采用**单进程多线程**的高性能极简架构。算法必须在 CI/CD 流水线中通过 ASan/LSan/TSan 的自动化拦截左移测试。算法上传后，Go 端通过 TCP 触发某个 C++ 节点主进程直接 `dlopen` 加载并进行一次完整推理自检。只有完成加载、推理且返回 JSON 格式符合 Schema 的算法版本才能进入可用状态。
- **热更新与引用计数**：
  不同算法版本解压至独立目录，各自持有 `dlopen` 句柄。新版本发布后，新增任务使用新版本；旧版本受引用计数保护，直到最后一个关联任务停止时才执行 `dlclose`，实现零停机热更新。

### 3.3 规则引擎与事件流转

为了避免后端慢 IO（存图、写库、网络推送）阻塞前置推理流，系统设计了异步数据流水线：

`Source (C++ TCP) -> Dispatcher (WebSocket) -> Filter (规则过滤) -> Deduplicator (去重) -> Transformer -> Sink (Webhook/DB)`

1. **Source**: C++ 吐出的原始 JSON 推理结果，附加时间戳和任务元数据。
2. **Dispatcher**: 无论是否产生告警，直接旁路分发一份数据至 WebSocket，供前端绘制 AI Overlay。
3. **Filter**: 根据任务配置的 ROI/MARK、置信度阈值、生效时间段进行内存级非阻塞过滤。
4. **Deduplicator**: 基于 `(TaskID + CategoryCode + TrackID)` 和时间窗口去重，防止告警风暴。
5. **Transformer**: 将原始元数据映射为告警事件，补充设备 SN 等上下文。
6. **Sink**: 分发至 Asynq 队列异步处理抓拍图保存、PG 数据库落库和 Webhook 重试推送。

### 3.4 视频流状态机与异常恢复

系统面对的是极度不稳定的边缘网络环境。

- **设备状态判定**：结合 ZLM `on_stream_changed` Webhook 回调、GB28181 注册心跳和轻量级 TCP 探活，综合计算设备的 online/offline 状态。
- **指数退避重连**：当拉流断开时，系统启动重连状态机。采用 5s、15s、30s 等指数退避策略重试，防止网络闪断恢复瞬间大规模并发重连引发信令雪崩。
- **ZLM 重启自愈**：当媒体服务崩溃重启后，Go 控制端捕获重启事件，自动重新向 GB28181 设备发起 Invite 点播，并将受影响任务置入重连队列。

## 4. 核心接口规范

### 4.1 算法 C API 接口定义

```cpp
#ifdef __cplusplus
extern "C" {
#endif

// 初始化探测器，传入解压后的算法包路径
void* create_detector(const char* package_dir);

// 执行单帧推理，返回堆分配的 JSON 字符串
char* detector_infer(void* detector,
                     const NikoImageFrame* image_array,
                     const char* ai_params_json);

// 释放推理结果字符串内存
void detector_free_result(char* result);

// 销毁探测器实例，释放 NPU 资源
void destroy_detector(void* detector);

#ifdef __cplusplus
}
#endif
```

### 4.2 统一类别编码规范 (Category Code)

模型内部产生的索引（如 `0, 1, 2`）缺乏全局语义。平台要求所有算法通过包内的 `label_map.json` 将结果转换为 5 位 `integer` 的系统类别编码。

- `10001`: 人员
- `10004`: 未佩戴安全帽
- `14001`: 区域入侵
- `11001`: 车牌

C++ 推理输出 JSON 示例：

```json
[
  {
    "category_code": 10001,
    "confidence": 0.985,
    "bbox": { "x": 100, "y": 200, "w": 50, "h": 80 },
    "matched_roi_ids": ["roi_001"]
  }
]
```

## 5. 数据存储架构

- **关系型数据 (PostgreSQL)**：
  - 高频表（如智能记录、告警记录）强制使用**按日期的表分区策略 (Partitioning)**。
  - 存储空间清理服务直接执行 `DROP PARTITION`，杜绝 `DELETE` 带来的表膨胀和锁等待。
- **向量检索 (pgvector)**：
  - 使用 PostgreSQL 的 `pgvector` 插件保存 512 维的人脸 Embedding。
  - 支持直接在数据库层面进行 1:N 余弦相似度 Top-K 检索。
- **文件存储 (Storage Abstraction)**：
  - 抓拍图片、人员底库图通过统一 Storage 接口落盘，支持配置挂载至 Local FS 或外部 OSS。

## 6. 系统容灾与防雪崩设计

1. **准入控制 (Admission Control)**：Go 端依据 NPU/内存阈值，拒绝超出硬件负荷的 `StartStream` 任务请求。
2. **推理队列丢帧策略**：C++ 端推理队列设限（如 30 帧）。当 NPU 算力不足以消费视频帧率时，主动丢弃最老帧，保证送入 NPU 的始终是最新画面。
3. **C++ 进程看门狗**：Go 与 C++ 通过 Ping-Pong 心跳监控。若 C++ 进程死锁或 OOM 崩溃，Go 端检测超时后将强制 Kill 进程、重启引擎，并全量下发运行中任务，实现微秒级业务自愈。
4. **Webhook 隔离与限流**：向第三方推送的告警全量进入 Asynq 队列，即使第三方宕机也不会阻塞边缘系统内存。队列积压严重时执行丢弃非核心事件降级。
