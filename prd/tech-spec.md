# Technical Specifications: AIVisionInference System

> 本文档为 AIVisionInference PRD 的技术详细规格附件。

### Architecture Overview

AIVisionInference 采用控制面与数据面分离架构。

```text
Admin Web / Business API
        |
        v
Go Management Plane
  - REST API / Auth / RBAC
  - Device Manager
  - GB28181 SIP Server / ZLM Client
  - Algorithm Package Manager
  - Task Scheduler
  - Event Router
  - Webhook Dispatcher
  - Storage Cleaner
        |
        | HTTP API / WebHook
        v
ZLMediaKit Media Service
  - RTSP / GB28181 / RTP Stream Ingest
  - WebRTC / HTTP-FLV / WS-FLV / HLS Output
  - Stream Status Callback
        |
        | RTSP / RTP stream for inference
        v
C++ Inference Engine
  - Stream Worker
  - RTSP Puller
  - Hardware Decoder Adapter
  - NPU Runtime Adapter
  - Algorithm Manager
  - Result Serializer
        |
        v
Hardware Backend: RK MPP / RKNN, DVPP / CANN, FFmpeg / ONNX Runtime
```

- **Go Management Plane**:
  - 提供 Admin API、业务 API、鉴权、RBAC 和审计日志。
  - 管理设备、分组、算法包、任务、人员、智能记录和存储策略。
  - 内置或集成 GB28181 SIP 服务端，支持国标设备注册、心跳、目录同步和点播控制。
  - 作为 ZLM 客户端调用 ZLM HTTP API，接收 ZLM WebHook 回调，并向前端签发可播放 URL。
  - 通过 UDS 向 C++ 发送控制命令，并接收结构化推理结果。
  - 将事件写入数据库、推送前端 WebSocket、触发 Webhook 上报。

- **C++ Inference Engine**:
  - 每路流由独立或池化 Worker 管理。
  - 推理输入流优先从 ZLM 暴露的本地 RTSP/RTP 地址获取，避免 C++ 引擎重复处理 GB28181、RTMP、HLS 等多协议接入细节。
  - 执行 `ZLM Stream Pull -> HW Decode -> NPU Infer -> Result JSON` 数据路径。
  - 管理算法动态库加载、引用计数、版本切换和资源释放。
  - 在 NPU 处理能力不足时支持丢帧策略：推理队列设置上限（默认 `30` 帧），新帧入队时若队列已满则丢弃最旧帧，始终保持最新帧可用；丢帧计数器累加并可通过可观测性接口查询。

### Integration Points

- **Frontend Admin**:
  - 用户、人员、设备、算法、任务、实时预览、智能记录、存储配置、系统配置页面。
  - 任务配置页面内置实时流播放器和 Canvas 区域编辑器，用于可视化绘制 ROI / MARK / LINE。
  - 实时预览播放地址由 Go 管理端基于 ZLM Stream ID 生成，前端仅消费 WebRTC、HTTP-FLV、WS-FLV、HLS 或 MP4-FMP4 等浏览器可播放地址。
  - 实时结果通过 WebSocket 或 SSE 推送。
- **Cache & Task Queue (Redis + Asynq)**:
  - 采用 Asynq 分布式任务队列处理高延迟或重试类任务，如：Webhook 的指数退避重试推送、定时的存储阈值清理、定时的设备在线状态探测拉流任务。
  - Redis 提供关键状态（如设备在线状态、任务运行态）的快速查询与缓存。
- **File Storage (Storage Abstraction)**:
  - 系统使用统一的 Storage 抽象层存储人员图片和实时抓拍图，支持配置切换 Local、PostgreSQL LOB 或 OSS（如 S3）。
- **Database (PostgreSQL + pgvector)**:
  - 存储用户、角色、设备、分组、算法包、任务、人员、事件记录、存储清理记录、系统配置和审计日志。
  - **表分区策略 (Table Partitioning)**：针对高频写入的智能记录表（识别、告警、抓拍），必须采用按时间（如按天）的表分区策略。存储空间清理时直接 Drop 最老的分区，确保数据库不发生表膨胀和死锁。
  - 人脸 Embedding 通过 **pgvector** 插件存储为 vector 类型，支持 1:N 余弦相似度 Top-K 查询。
  - 人员图片由系统在导入时自动提取 Embedding 并写入 pgvector 索引。
- **ZLMediaKit Media Service**:
  - 作为系统内部媒体服务进程部署，负责视频流接入、协议转换、流保活、按需拉流、播放 URL 生成和 WebHook 状态回调。
  - 支持 RTSP 设备拉流、GB28181 设备点播后推流、RTP/PS 流接入，并输出 WebRTC、HTTP-FLV、WS-FLV、HLS 或 MP4-FMP4。
  - Go 管理端保存 ZLM `app`、`stream`、`schema`、`vhost` 等流标识映射，统一关联平台 `device_id` 和 `task_id`。
- **C++ Engine IPC**:
  - Go 与 C++ 使用 Unix Domain Socket 通信。
  - 控制面传递轻量命令和参数，不传递大规模视频帧。
- **Third-party Webhook**:
  - 告警事件通过 RESTful POST 推送至第三方平台。
  - 支持自定义请求头、指数退避重试和基于 `event_id` 的幂等去重。

### Internationalization (i18n) Architecture

- **后端机制**: Go 端通过请求头 `Accept-Language` 确定当前语言。所有业务错误返回预定义的 `error_code` 和对应的 `message_key`。底层错误（如数据库、系统调用）必须被封装，不得直接抛给前端。
- **字典管理**: 翻译字典（JSON/YAML）统一定义在 Go 后端和前端代码中。后端根据字典将 `message_key` 翻译为目标语言的 `message` 返回。
- **前端适配**: 前端使用 `i18next` 或 `vue-i18n` 进行界面静态文本翻译。对于后端返回的动态错误信息，优先展示 `message`，若为空则前端基于 `message_key` 兜底翻译。

### Edge Cases & Boundary Conditions

- **系统时间回退**: 当 NTP 同步导致系统时间发生回退时，可能影响智能记录的时间戳顺序或触发定时任务异常。系统应在时间回退时记录高风险审计日志，并在生成 `event_id` 时采用不完全依赖时间的 UUIDv4 机制，避免主键冲突。
- **并发操作与数据竞争**: 多个管理员同时修改同一任务或设备配置时，系统应通过数据库的乐观锁（如 `updated_at` 或 `version` 字段）防止丢失更新 (Lost Update)。
- **Token 刷新策略**: 前端应在 JWT Token 即将过期前（如提前 5 分钟）自动调用刷新接口获取新 Token，避免用户在填写长表单（如任务配置）时因过期被强制登出。
- **进程级 OOM 恢复**: 若 C++ 推理引擎因模型内存泄漏导致 OOM 被系统 Kill，Go 端进程管理模块应在 3 秒内自动拉起新的 C++ 进程，并自动重新下发所有处于 `running` 状态的任务，实现业务自愈。

### Protobuf Message Definition & IPC

Go 与 C++ 之间需要保证强类型、低延迟和流式双向通信。为避免嵌入式设备（如 RK3576 / Ascend）引入庞大的 gRPC 依赖，并实现极客级别的零拷贝，系统采用 **Raw UDS (Unix Domain Socket) + Length-Prefixed Protobuf + SCM_RIGHTS (FD 传递)** 的通信架构。

#### 1. 通信通道分离

1. **Control Channel (控制通道)**: 处理 `StartTask`、`StopTask`、`Heartbeat` 等控制信令。
2. **Data/Event Channel (数据通道)**: C++ 通过 UDS 持续向 Go 推送结构化的推理结果 (`InferenceResult`) 和系统事件 (`SystemEvent`)。
3. **Zero-Copy Shared Memory (单图零拷贝)**: 针对单图推理，Go 端通过 `memfd_create` 创建匿名内存写入图片，通过 UDS 外带数据 (SCM_RIGHTS) 传递 File Descriptor (FD) 给 C++。C++ 直接 `mmap` 读取内存进行推理，完全消除 Socket 缓冲区拷贝。

#### 3. 协议帧格式 (Wire Format)

数据在 UDS 字节流中的格式为：
`[ 4 Bytes Length (Big-Endian) ] + [ Protobuf 二进制数据 (IpcEnvelope) ]`

#### 4. 核心 Protobuf 定义 (`inference.proto`)

```protobuf
syntax = "proto3";

package aivision.ipc;
option go_package = "./pb";

// 顶层信封，用于多路复用
message IpcEnvelope {
    enum MsgType {
        UNKNOWN = 0;
        // 控制请求 (Go -> C++)
        CMD_PING = 1;
        CMD_START_TASK = 2;
        CMD_STOP_TASK = 3;
        CMD_INFER_SINGLE_IMAGE = 4; // 单图推理请求 (附带 FD)

        // 事件推送 (C++ -> Go)
        EVT_INFERENCE_RESULT = 10;
        EVT_SYSTEM_EVENT = 11;

        // 统一响应 (C++ -> Go)
        ACK_RESPONSE = 20;
    }

    MsgType type = 1;
    uint64 sequence_id = 2; // 请求序号，Go端生成，C++ ACK时原样返回，用于请求-响应匹配
    bytes payload = 3;      // 具体的业务载荷(序列化后的具体Request/Response/Event)

    // 附加信息：如果当前消息携带了 FD，这里记录该 FD 对应的数据大小
    int64 fd_payload_size = 4;
}

// ------------------------------------------
// 基础控制与任务配置
// ------------------------------------------
message Pong {
    int64 timestamp = 1;
    float npu_usage = 2; // C++端负载，供Go做准入控制
    float mem_usage = 3;
}

message StartTaskRequest {
    string task_id = 1;
    string stream_url = 2;              // RTSP/GB28181等媒体流地址
    int32 decode_hw_type = 3;           // 0:Auto, 1:RKMPP, 2:DVPP, 3:FFmpeg
    repeated AlgoConfig algorithms = 4; // 该任务绑定的算法列表(串/并联)
}

message AlgoConfig {
    string algo_name = 1;
    string so_path = 2;                 // 动态库绝对路径
    string params_json = 3;             // 动态运行参数(由algo_meta.yaml定义)

    // 区域与越界线配置 (归一化坐标 0.0~1.0)
    repeated Region roi_regions = 4;
    repeated Region mark_regions = 5;
    repeated Line trip_lines = 6;
}

message Region {
    string region_id = 1;
    repeated Point points = 2;
}

message Line {
    string line_id = 1;
    Point start = 2;
    Point end = 3;
    int32 direction = 4; // 0: 双向, 1: A->B, 2: B->A
}

message Point {
    float x = 1;
    float y = 2;
}

message StandardResponse {
    int32 error_code = 1;       // 0表示成功
    string error_message = 2;   // 内部错误信息(Go端负责i18n)
}

// ------------------------------------------
// 推理结果与底层异常事件
// ------------------------------------------
message InferenceResult {
    string task_id = 1;
    int64 frame_pts = 2;
    string algo_name = 3;
    repeated Object objects = 4;
}

message Object {
    int32 class_id = 1;
    string class_name = 2;
    float confidence = 3;
    int32 track_id = 4;

    float bbox_x = 5; float bbox_y = 6;
    float bbox_w = 7; float bbox_h = 8;

    string extra_data_json = 9; // 算法自定义额外输出
}

message SystemEvent {
    string task_id = 1;
    enum EventLevel { INFO = 0; WARN = 1; ERROR = 2; FATAL = 3; }
    EventLevel level = 2;
    string event_code = 3;      // 预定义事件码: STREAM_DISCONNECT, ALGO_HUNG, OOM_WARNING
    string message = 4;
}
```

#### 5. 零拷贝与资源管理约束

- **单图推理共享内存**: Go 端通过 `memfd_create(MFD_CLOEXEC)` 创建匿名内存，利用 `unix.Sendmsg` 发送带有 `SCM_RIGHTS` 标志的控制报文。C++ 端通过 `recvmsg` 提取 FD，利用 `mmap(PROT_READ, MAP_PRIVATE)` 直接读取数据。
- **FD 泄漏监控**: 由于 FD 指向的是匿名内存，C++ 端如果发生未捕获异常或忘记 `close(fd)`，将导致严重的物理内存泄漏（OOM）。C++ 侧必须使用 RAII 模式严格管理 `fd` 的 `close` 与内存的 `munmap`。
- **粘包处理**: UDS 面向字节流，C++ 端读取时必须先严格 `recv` 4 字节 Header，解析出 Protobuf payload 长度，再循环读取直至满帧，防止 TCP 粘包/半包问题。注意 FD (辅助数据) 通常仅依附在携带帧起始数据的第一个报文中。
- **媒体流架构**: RTSP/GB28181 流的字节不通过 Go 端转发，而是由 C++ 推理引擎内嵌的 client 直接到媒体源或内部 ZLM 拉流，彻底实现控制面与数据面分离。

### ROI, MARK & LINE Configuration

系统必须支持在任务维度配置 ROI、MARK 和 LINE，并将配置作为 `ai_param_json` 的标准字段下发给 C++ 推理引擎。

#### ROI Definition

ROI（Region of Interest）表示算法生效区域。推理引擎应根据配置对算法输出结果进行过滤，只保留 ROI 内的目标，或在结果中标记目标是否命中 ROI。

```json
{
  "roi_regions": [
    {
      "id": "roi_001",
      "name": "入口区域",
      "enabled": true,
      "type": "polygon",
      "color": "#22C55E",
      "points": [
        { "x": 0.10, "y": 0.20 },
        { "x": 0.80, "y": 0.20 },
        { "x": 0.75, "y": 0.85 },
        { "x": 0.12, "y": 0.82 }
      ]
    }
  ]
}
```

#### MARK Definition

MARK 表示掩码区域（Mask Area），用于配置算法屏蔽区域。落入 MARK 掩码区域的目标默认不参与结果输出、告警判断或统计计算。

```json
{
  "mark_regions": [
    {
      "id": "mark_001",
      "name": "屏蔽区域",
      "enabled": true,
      "type": "polygon",
      "color": "#F97316",
      "points": [
        { "x": 0.20, "y": 0.30 },
        { "x": 0.85, "y": 0.30 },
        { "x": 0.85, "y": 0.65 },
        { "x": 0.20, "y": 0.65 }
      ]
    }
  ]
}
```

#### LINE Definition

LINE 表示越界配置，用于配置越线检测、方向判断或区域边界越界。LINE 与 MARK 不同：MARK 是掩码屏蔽区域，LINE 是业务判定边界。

```json
{
  "line_regions": [
    {
      "id": "line_001",
      "name": "入口越界线",
      "enabled": true,
      "type": "line",
      "color": "#3B82F6",
      "points": [
        { "x": 0.20, "y": 0.50 },
        { "x": 0.85, "y": 0.50 }
      ],
      "direction": "left_to_right",
      "trigger": "cross_line"
    }
  ]
}
```

#### Coordinate & Validation Rules

- ROI / MARK / LINE 坐标统一使用归一化坐标，`x` 和 `y` 取值范围均为 `0.0` 到 `1.0`。
- ROI 与 MARK 使用 `type=polygon`，`points` 至少包含 `3` 个点。
- LINE 使用 `type=line`，`points` 必须包含 `2` 个点。
- LINE 的 `direction` 用于声明触发方向，允许值包括 `any`、`left_to_right`、`right_to_left`、`top_to_bottom`、`bottom_to_top`。
- LINE 的 `trigger` 用于声明触发类型，允许值包括 `cross_line`、`enter_region`、`leave_region`。
- 同一任务内 ROI ID、MARK ID 和 LINE ID 必须唯一。
- 禁用状态的 ROI / MARK / LINE 仍可保存，但不得参与算法判定。
- ROI 用于限定算法生效区域；MARK 用于定义算法屏蔽区域；LINE 用于定义越界判定边界。
- 当目标同时命中 ROI 和 MARK 时，MARK 优先级高于 ROI，目标默认应被过滤。
- 任务配置页面必须基于实时流画面提供 Canvas 绘制层，用户在播放画面上直接绘制 ROI / MARK / LINE。
- 前端绘制时应基于原始视频宽高进行归一化转换；显示时再按当前播放器尺寸还原。
- 当实时流暂时不可用时，允许使用最近一帧截图继续编辑，但保存时需提示用户当前画面可能不是最新实时画面。
- Go 端负责保存、校验和下发 ROI / MARK / LINE 配置；C++ 推理引擎负责按配置执行 ROI 生效区域过滤、MARK 掩码区域过滤和 LINE 越界判定。

#### `ai_param_json` Example

```json
{
  "conf_thres": 0.5,
  "iou_thres": 0.45,
  "roi_regions": [],
  "mark_regions": [],
  "line_regions": []
}
```

每个算法配置维护各自的 `ai_param_json`，即每个算法可以拥有自己独立的阈值参数和 ROI/MARK/LINE 空间区域设置。

### Hardware Decode & Zero-copy Abstraction

C++ 端封装统一硬解接口 `IDecoder`。

```cpp
class IDecoder {
public:
    virtual ~IDecoder() = default;
    virtual bool Open(const std::string& rtsp_url) = 0;
    virtual DecodeFrame ReadFrame() = 0;
    virtual void Close() = 0;
};
```

- Rockchip 平台优先使用 MPP 输出 `dma_buf` 或硬件可访问内存句柄。
- Huawei Ascend 平台优先使用 DVPP 输出可被 CANN 处理的设备内存。
- FFmpeg 作为软解降级方案，用于开发、调试或不支持硬解的平台。
- 解码输出帧应尽量封装为 `NikoImageFrame`，并通过 `NikoNikoDetector::infer(image_array, ai_params)` 传入算法包，避免在接口层暴露过多离散参数，同时支持平台推理适配层直接传递设备内存句柄以减少 CPU 用户态拷贝。

### Algorithm Package Standard

算法包以目录或压缩包形式交付，至少包含：

```text
algorithm-package/
  algo_meta.yaml
  nikoniko_detector.so  # 必需，算法入口动态库
  testimage.jpg            # 必需，上传自检推理图片
  label_map.json           # 必需，固定命名，类别索引/类型 ID 到系统类别编码的映射
  models/                  # 可选，若动态库需要外部模型文件
  README.md                # 可选，说明算法能力和参数含义
```

#### `algo_meta.yaml` Example

```yaml
algorithm: "yolov8_person_det"
version: "1.2.0"
domain: "generic_object"
result_schema: "object_detection"
capabilities:
  image: ["detect"]
  data: ["track"]
description: "高精度行人检测算法，适用于监控场景"
hardware: ["rk3588", "rk3576"]

ai_params_schema:
  type: "object"
  properties:
    conf_thres:
      type: "number"
      title: "置信度阈值"
      description: "过滤低置信度目标的阈值"
      default: 0.5
      minimum: 0.0
      maximum: 1.0
    iou_thres:
      type: "number"
      title: "NMS IOU 阈值"
      description: "非极大值抑制重叠度阈值"
      default: 0.45
      minimum: 0.0
      maximum: 1.0
    enable_tracker:
      type: "boolean"
      title: "启用目标追踪"
      description: "在视频流模式下启用目标追踪算法"
      default: true
  required: ["conf_thres", "iou_thres"]
```

> **注**: `algo_meta.yaml` 中仅声明该算法专属的动态参数。通用的 ROI、MARK、LINE 配置由 Go 管理端在表单渲染时自动注入，无需在 YAML 中重复声明。

#### Fixed Package Convention

算法包采用统一固定规范，`algo_meta.yaml` 中不需要声明 `runtime`、`entrypoint`、`class_name` 或 `label_map`。

固定约定如下：

- `runtime` 固定为 `cpp`。
- 算法入口动态库固定为 `nikoniko_detector.so`。
- 动态库内部类名固定为 `NikoNikoDetector`。
- 类别映射文件固定为 `label_map.json`。
- 自检图片固定为 `testimage.jpg`。
- 初始化参数不对外暴露；模型路径、运行设备和厂商 Runtime 参数由算法包内部自行定义。

上传自检、任务启动和 C++ 推理服务均按上述固定文件名和固定类名加载算法包。

同名动态库隔离约定：

- 所有算法包内的入口动态库都可以同名为 `nikoniko_detector.so`。
- 每个算法包必须解压到独立目录，例如 `algorithms/{algorithm}/{version}/nikoniko_detector.so`。
- C++ 推理服务必须使用动态库的绝对路径执行 `dlopen`，不得只按文件名从全局搜索路径加载。
- `dlopen` 建议使用 `RTLD_NOW | RTLD_LOCAL`，避免不同算法包之间的符号污染。
- 算法包升级时，新旧版本应位于不同版本目录，通过引用计数控制旧版本卸载，不通过覆盖同一路径文件完成热更新。
- 因此不同算法包使用相同文件名不会互相影响，实际隔离边界是算法包目录和版本目录。

#### Label Map Standard

算法模型通常只输出类别索引、类别 ID 或内部标签。为了让平台能够统一统计、告警和展示，算法包必须提供固定命名的 `label_map.json`，将模型输出值映射为系统定义的类别编码。系统类别编码必须使用五位数字编码，不允许使用字符串编码。

`label_map.json` 示例：

```json
[
  {
    "label_id": 0,
    "category_code": 10001,
    "display_name": "人员"
  },
  {
    "label_id": 1,
    "category_code": 10002,
    "display_name": "车辆"
  },
  {
    "label_id": 2,
    "category_code": 10003,
    "display_name": "已佩戴安全帽"
  }
]
```

映射约束：

- 映射文件名固定为 `label_map.json`。
- `label_id` 必须与模型内部输出的类别索引、类型 ID 或内部标签可一致匹配。
- `category_code` 必须使用平台预定义的五位数字类别编码，类型为 integer，取值范围为 `10000` 到 `99999`。
- `display_name` 用于后台展示，可选；前端最终展示文案仍需支持 i18n。
- 若算法内部推理结果出现无法映射的类别索引/类型 ID，上传自检必须失败。
- C++ 算法包必须在返回给 Go 端之前完成映射，最终结果中不得返回模型内部 `class_id`、`type_id` 或 `label_id`，而应直接返回五位数字 `category_code`。
- Go 端接收到 C++ 推理结果后，只根据 integer 类型的五位数字 `category_code` 查询数据库中的系统类别定义，不再负责模型索引到类别编码的映射。

#### Algorithm Runtime Interface

算法包必须以 C++ 动态库形式交付。动态库内部必须实现 `NikoNikoDetector` 类；为了避免 C++ ABI、name mangling、标准库版本和跨 `.so` 对象析构问题，动态库对外不得要求主服务直接链接 C++ 类符号，而必须额外导出稳定的 C ABI Factory 函数。

##### Internal C++ Class

```cpp
struct NikoImageFrame {
    const unsigned char* data;
    int width;
    int height;
    int channels;
    const char* pixel_format;
    int stride;
    int dma_fd;      // 可选，设备内存句柄；无设备内存时为 -1
    void* user_data; // 可选，厂商 Runtime 适配层扩展数据
};

class NikoNikoDetector {
public:
    explicit NikoNikoDetector(const char* package_dir);

    ~NikoNikoDetector();

    std::string infer(const NikoImageFrame* image_array,
                      const char* ai_params_json);
};
```

职责约束：

- `NikoImageFrame` 用于统一封装图片数组及其元信息，避免 `infer` 接口暴露宽、高、通道、像素格式等多个离散参数。
- `NikoNikoDetector` 类名固定，大小写固定。
- 构造函数负责加载模型、初始化厂商 Runtime、选择 NPU/GPU/CPU 等运行设备，并创建算法上下文；这些初始化细节由算法包内部自行定义，不允许用户动态配置。
- `infer` 负责执行单次图片数组推理。
- `image_array` 表示当前待推理图片或视频帧对象，类型为 `NikoImageFrame`，内部封装图像数据指针、宽高、通道数、像素格式、stride、设备内存句柄等元信息。
- `ai_params_json` 必须与 `algo_meta.yaml` 中的 `ai_params_schema` 保持兼容。
- `infer` 返回值必须是 JSON 字符串，内容为可反序列化的 `list[dict]`，结构需符合本 PRD 的 Output JSON Schemas。
- 当模型输出类别索引或类型 ID 时，算法包内部必须通过 `label_map.json` 转换为系统统一五位数字类别编码；返回给 Go 端的结果必须包含 integer 类型的 `category_code`，不得暴露模型内部 `class_id`、`type_id` 或 `label_id`。

##### Exported C ABI Factory

```cpp
#ifdef __cplusplus
extern "C" {
#endif

void* create_detector(const char* package_dir);

char* detector_infer(void* detector,
                     const NikoImageFrame* image_array,
                     const char* ai_params_json);

void detector_free_result(char* result);

void destroy_detector(void* detector);

#ifdef __cplusplus
}
#endif
```

C ABI 约束：

- `create_detector` 创建并返回 `NikoNikoDetector` 实例指针，`package_dir` 为算法包解压后的绝对目录路径，仅用于算法包定位模型文件和内部资源，不作为用户动态配置参数；失败时返回 `nullptr`，错误信息由推理引擎捕获并记录。
- `detector_infer` 调用内部 `NikoNikoDetector::infer`，返回堆分配的 JSON 字符串。
- `detector_free_result` 必须释放 `detector_infer` 返回的字符串内存，避免跨动态库内存释放风险。
- `destroy_detector` 销毁 `NikoNikoDetector` 实例并释放模型、显存/NPU 资源、线程和句柄。
- C++ 推理服务必须通过 `dlopen` / `dlsym` 加载上述 C ABI 符号，不直接依赖 C++ 类 ABI。
- 算法内部异常必须在 C ABI 边界内捕获，转换为明确错误结果或空指针返回，不得让异常跨 `.so` 边界传播。

#### Upload Self-check

算法包上传时必须执行一次自检，确保该算法包具备最小可运行能力。

自检输入：

- `testimage.jpg`: 算法包根目录下的必需文件。
- `label_map.json`: 算法包根目录下固定命名的必需类别映射文件。
- `ai_params`: 根据 `ai_params_schema` 生成的默认推理参数；如字段存在 `default` 则使用默认值。

自检流程：

1. 解压算法包到隔离临时目录。
2. 校验 `algo_meta.yaml`、固定入口动态库 `nikoniko_detector.so`、`testimage.jpg` 和 `label_map.json` 是否存在。
3. 通过 `dlopen` 加载固定入口动态库 `nikoniko_detector.so`。
4. 通过 `dlsym` 校验并获取 `create_detector`、`detector_infer`、`detector_free_result`、`destroy_detector` 四个函数。
5. 调用 `create_detector(package_dir)` 初始化 `NikoNikoDetector` 实例。
6. 读取 `testimage.jpg`，解码并封装为 `detector_infer` 可接收的 `NikoImageFrame`。
7. 调用 `detector_infer` 执行一次推理。
8. 解析返回 JSON，并校验结果结构符合本 PRD 的 Output JSON Schemas。
9. 校验返回结果必须包含 integer 类型的五位数字 `category_code`，且不得包含模型内部 `class_id`、`type_id` 或 `label_id`。
10. 校验 `category_code` 属于类别映射文件中定义的系统统一五位数字类别编码。
11. 调用 `detector_free_result` 和 `destroy_detector` 释放资源。

自检通过条件：

- 动态库可成功加载。
- 四个 C ABI Factory 函数均存在。
- `NikoNikoDetector` 可成功初始化。
- `testimage.jpg` 可成功读取并解码。
- 类别映射文件可成功读取并解析。
- `detector_infer` 调用成功。
- 返回值是合法 JSON。
- 返回 JSON 的顶层结构为 `list[dict]`，且字段类型符合对应 `result_schema` 的 Output JSON Schemas。
- 推理结果中直接包含 integer 类型的五位数字 `category_code`，且该编码存在于类别映射文件的 `category_code` 集合中。
- 推理结果中不得包含模型内部 `class_id`、`type_id` 或 `label_id`。

自检不负责判断：

- 检测框是否准确。
- 分类标签是否正确。
- 人脸身份是否匹配。
- 置信度是否达到业务阈值。
- 模型精度、召回率、误报率等质量指标。

### Output JSON Schemas

C++ 算法包只返回核心推理结果数组。若模型输出为类别索引、类型 ID 或内部标签，C++ 算法包必须根据 `label_map.json` 在算法包内部完成映射，返回给 Go 端的结果必须直接包含系统统一五位数字类别编码 `category_code`，类型为 integer。Go 端不负责模型索引映射，只根据 `category_code` 查询数据库中的类别定义，并结合任务上下文、算法元数据、ROI / MARK / LINE 配置、设备信息和业务规则组装最终响应或事件记录。

若算法结果与 ROI / MARK / LINE 相关，结果中可返回 `matched_roi_ids` 记录目标命中的 ROI。正常结果不应包含落入 MARK 掩码区域的目标；如需调试，可返回 `masked_by_mark_ids` 表示目标被哪些 MARK 掩码区域过滤。若触发越界规则，可返回 `triggered_line_ids` 和 `direction`。

坐标体系统一规则：ROI / MARK / LINE 配置使用归一化坐标（`0.0` 到 `1.0`）；算法输出中的 `bbox`、`polygon`、`landmarks`、`contours` 和 `char_boxes` 默认使用原始视频帧像素坐标。C++ 后处理在执行 ROI / MARK / LINE 判定时，必须基于原始帧宽高将归一化区域配置转换为像素坐标；前端 Overlay 展示时再按播放器当前尺寸进行缩放映射。

#### Algorithm Metadata Taxonomy

算法不能只用单一 `type` 分类，因为一个业务算法可能同时包含多种能力。例如车牌识别通常包含目标检测、OCR、颜色分类和结构化后处理；人脸识别通常包含检测、关键点、Embedding 和身份比对。因此算法元数据采用三层描述：

1. `domain`: 业务领域，描述算法面向的业务对象或场景。
2. `result_schema`: 输出结构，描述算法返回结果应遵循哪一种 JSON Schema。
3. `capabilities`: 按数据变换签名分类的能力列表，描述算法内部具备哪些变换能力。`capabilities` 分为 `image`（输入为图像）和 `data`（输入为结构化数据）两组，一个算法可同时声明多组能力。

##### Domain Examples

- `generic_object`: 通用目标。
- `person`: 人员。
- `vehicle`: 车辆。
- `license_plate`: 车牌。
- `face`: 人脸。
- `text`: 文本。
- `safety`: 安全生产。
- `traffic`: 交通事件。
- `custom`: 自定义业务场景。

##### Result Schema Examples

- `object_detection`: 返回目标框、类别编码和置信度。
- `image_classification`: 返回分类类别编码和置信度。
- `face_identity`: 返回人脸框、关键点、Embedding 或身份信息。
- `instance_segmentation`: 返回目标轮廓、mask 或分割区域。
- `structured_text`: 返回文本、字符框、文本置信度。
- `license_plate`: 返回车牌号、车牌颜色、车牌类型、车牌框和字符框。
- `keypoints`: 返回关键点坐标及置信度。
- `event`: 返回事件编码、事件状态和触发区域。

##### Capability Examples

###### 1. Core Vision Capabilities (`image` 组: Image → Structured Data)

这些能力以图像或图像区域为输入，输出结构化数据。

| 标识符 | 输入 | 输出 | 说明 |
|---|---|---|---|
| `detect` | Image | `[{bbox, category_code, confidence}]` | 目标检测 |
| `classify` | Image / Crop | `{category_code, confidence}` | 图像分类 |
| `segment` | Image | `[{contours/mask, category_code, confidence}]` | 实例/语义分割 |
| `ocr` | Image / Crop | `[{text, bbox, confidence}]` | 文本识别 (OCR) |
| `estimate_keypoints` | Image | `[{keypoints, confidence}]` | 关键点检测 |
| `extract_embedding` | Image / Crop | `[float]` | 特征向量提取 |

###### 3. Structured Data Capabilities (`data` 组: Structured Data → Enriched Data)

这些能力以结构化数据为输入（可能需要多帧上下文或外部特征库），输出更高级别的结构化数据。

| 标识符 | 输入 | 输出 | 说明 |
|---|---|---|---|
| `track` | `[{bbox, frame_id}]` | `[{track_id, bbox}]` | 跨帧目标追踪 |
| `recognize_identity` | `[float]` + 特征库 | `{identity_id, confidence}` | 1:N 身份比对 |

##### Standard Post-processing Rules（引擎内置标准后处理）

以下规则为推理引擎内置的标准后处理能力，不属于算法包的 `capabilities`。系统根据任务配置中的 ROI / MARK / LINE 设置自动作用于算法输出。

| 规则标识符 | 作用层级 | 输入依赖 | 说明 |
|---|---|---|---|
| `roi_filter` | 推理结果过滤 | 算法输出包含 `bbox` | 只保留 ROI 区域内的目标 |
| `mark_mask` | 推理结果过滤 | 算法输出包含 `bbox` | 排除落在 MARK 掩码区域的目标 |
| `line_crossing` | 追踪轨迹判断 | 算法具备 `track` 能力 | 基于轨迹判断越界/越线事件 |
| `anomaly_judge` | 结果逻辑判定 | 算法输出包含 `category_code` | 基于业务规则判断异常状态 |

#### C++ Algorithm Result

```json
[
  {
    "category_code": 10001,
    "confidence": 0.985,
    "bbox": { "x": 100, "y": 200, "w": 50, "h": 80 },
    "matched_roi_ids": ["roi_001"],
    "masked_by_mark_ids": [],
    "triggered_line_ids": ["line_001"],
    "direction": "left_to_right",
    "attributes": {
      "age": 25,
      "gender": "male"
    }
  }
]
```

#### Go External Response

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "task_001",
    "device_id": "device_001",
    "algorithm": "yolov8_person_det",
    "algorithm_version": "1.2.0",
    "timestamp": "2026-06-03T10:00:00Z",
    "results": [
      {
        "category_code": 10001,
        "confidence": 0.985,
        "matched_roi_ids": ["roi_001"],
        "masked_by_mark_ids": [],
        "triggered_line_ids": ["line_001"],
        "direction": "left_to_right"
      }
    ]
  }
}
```

#### Detection Result

```json
[
  {
    "category_code": 10001,
    "confidence": 0.985,
    "bbox": {
      "x": 100,
      "y": 200,
      "w": 50,
      "h": 80
    },
    "matched_roi_ids": ["roi_001"],
    "masked_by_mark_ids": [],
    "triggered_line_ids": ["line_001"],
    "direction": "left_to_right"
  }
]
```

#### Classification Result

```json
[
  {
    "category_code": 10003,
    "confidence": 0.952
  }
]
```

#### Face Recognition Result

```json
[
  {
    "category_code": 13001,
    "identity_id": "employee_001",
    "similarity": 0.923,
    "detect_confidence": 0.981,
    "bbox": {
      "x": 120,
      "y": 150,
      "w": 40,
      "h": 40
    },
    "landmarks": [
      { "x": 130, "y": 160 },
      { "x": 145, "y": 160 },
      { "x": 137, "y": 170 }
    ],
    "matched_roi_ids": ["roi_001"],
    "masked_by_mark_ids": [],
    "triggered_line_ids": []
  }
]
```

> **注**: 普通人脸识别事件不返回 `embedding`，避免敏感特征泄露和结果包体过大。Embedding 仅允许在人员图片导入、特征提取任务或受权限控制的内部接口中生成并写入向量库。

#### Plate Recognition Result

用于车牌检测、车牌号识别、车牌颜色识别和车牌类型识别。建议元数据：`domain: "license_plate"`，`result_schema: "license_plate"`，`capabilities: { image: ["detect", "ocr", "classify"] }`

```json
[
  {
    "category_code": 11001,
    "plate_no": "粤B12345",
    "plate_color": "blue",
    "plate_type": "small_vehicle",
    "confidence": 0.972,
    "bbox": {
      "x": 320,
      "y": 240,
      "w": 180,
      "h": 48
    },
    "text_confidence": 0.961,
    "char_boxes": [
      { "text": "粤", "x": 326, "y": 244, "w": 22, "h": 38 },
      { "text": "B", "x": 352, "y": 244, "w": 20, "h": 38 }
    ],
    "matched_roi_ids": ["roi_001"],
    "masked_by_mark_ids": [],
    "triggered_line_ids": [],
    "direction": "any"
  }
]
```

#### OCR Result

用于通用文本检测与识别。建议元数据：`domain: "text"`，`result_schema: "structured_text"`，`capabilities: { image: ["detect", "ocr"] }`

```json
[
  {
    "category_code": 12001,
    "text": "安全出口",
    "confidence": 0.943,
    "bbox": {
      "x": 100,
      "y": 80,
      "w": 160,
      "h": 40
    },
    "polygon": [
      { "x": 100, "y": 80 },
      { "x": 260, "y": 82 },
      { "x": 258, "y": 120 },
      { "x": 102, "y": 118 }
    ]
  }
]
```

#### Segmentation Result

```json
[
  {
    "category_code": 10002,
    "confidence": 0.881,
    "bbox": { "x": 10, "y": 20, "w": 100, "h": 50 },
    "matched_roi_ids": ["roi_001"],
    "masked_by_mark_ids": [],
    "triggered_line_ids": [],
    "direction": "any",
    "contours": [
      { "x": 10, "y": 20 },
      { "x": 100, "y": 20 },
      { "x": 110, "y": 70 },
      { "x": 10, "y": 70 }
    ],
    "mask": "base64_or_rle_string"
  }
]
```

### Security & Privacy

- **Authentication & Authorization**:
  - 管理后台和业务 API 必须启用 JWT 鉴权。
  - 所有管理操作受 RBAC 控制。
- **Credential Protection**:
  - RTSP 账号密码、Webhook Token 等敏感信息需加密存储或脱敏展示。
  - 日志中禁止输出密码、Token、完整 RTSP 密码段。
- **Data Privacy**:
  - 人脸图片、人员信息、Embedding 属于敏感数据，需限制访问权限并记录访问审计。
  - 导出数据需受权限控制。
- **File Upload Security**:
  - 算法包上传需校验文件类型、大小、目录穿越风险、动态库入口函数、`testimage.jpg` 和类别映射文件是否存在。
  - 算法包入库前必须完成上传自检；自检失败的算法包不得启用。
  - 不允许算法包覆盖系统非目标目录文件。
- **Runtime Isolation**:
  - 算法动态库运行在 C++ 推理进程内，需通过进程级隔离降低崩溃对 Go 管理端影响。
  - C++ 推理进程异常退出后，Go 端应记录错误并自动重启 C++ 进程。
  - **双向心跳监控 (Heartbeat)**：Go 与 C++ 之间必须实现轻量级 Ping-Pong 心跳协议。若 Go 端连续超时未收到 C++ 的心跳响应，需判定 C++ 进程死锁或假死，应主动触发 Kill 操作并执行崩溃恢复流程。
  - Go 端维护任务状态持久化视图，C++ 进程重启后，Go 端自动将当前 `running` 状态的任务通过 UDS 重新下发 `StartStream` 命令，完成任务自动恢复。
  - C++ 进程崩溃时正在推理中的结果直接丢失，不补偿重推，由业务侧根据断流感知自行处理。
- **Internationalization**:
  - 返回给前端展示的错误消息必须支持 i18n，不直接暴露底层原始错误。
  - Go API 统一返回 `code`、`message` 和可选 `message_key`；`message` 根据用户语言配置或 `Accept-Language` 生成，`message_key` 用于前端兜底翻译。
  - C++ 推理服务只返回稳定的内部 `error_code` 和脱敏后的 `error_message`，Go 管理端负责将其映射为平台统一错误码与 i18n 文案。

### 4.9 System-level Boundary & Disaster Recovery

异构系统最怕组件“失联”，针对边缘计算场景的高压与恶劣网络环境，必须明确各个组件的“爆炸半径”和自愈策略。

#### 1. 进程级崩溃与恢复 (Process Crash & Recovery)

| 异常组件 | 触发边界/场景 | 爆炸半径与现象 | 系统自愈与降级策略 |
| :--- | :--- | :--- | :--- |
| **C++ 推理引擎崩溃** | 某算法 `.so` 内存泄漏 / 数组越界导致 Segfault | 所有正在运行的 AI 任务中断，但不影响 Go 管理端和流媒体播放。 | 1. Go 端探活（如 UDS ping 超时）检测到引擎下线；<br>2. Go 端自动拉起新的 C++ 引擎进程（最大重试次数配置，如防雪崩 3次/分钟）；<br>3. Go 端重新下发所有 `running` 状态的任务配置。 |
| **ZLMediaKit 崩溃** | 并发过载 / FLV 封装库异常导致进程退出 | 实时预览画面卡住，RTSP 拉流中断。 | 1. 守护进程（Supervisor / systemd / 容器探测）秒级拉起 ZLM；<br>2. Go 端接收到 ZLM 重启事件后，自动将挂载在 ZLM 的流及相关任务置为重连状态。 |
| **Go 管理端重启** | 升级部署 / OOM 被 kill / 系统断电重启 | API 无法访问，GB28181 信令断开。 | 1. Go 重启后，**必须从数据库恢复状态，不得丢失已配置任务**；<br>2. 重新建立与 ZLM 的通信，检测丢失的流；<br>3. 向 C++ 推理引擎全量下发任务状态（或如果 C++ 也重启，则重新拉起全链路）。 |
| **算法动态库 (`.so`) 卡死** | 模型 `infer()` 死循环 / 厂商 Runtime (如 RKNN) GPU 驱动挂起 | 某个任务长达数秒没有推理结果返回（不 Crash 但 Hung）。 | C++ 引擎内部应设置单帧推理 Watchdog。若 `infer()` 调用阻塞超过阈值（如 `2000ms`），强制终止该线程或该任务句柄，并向 Go 上报 `TaskHung` 异常事件，释放 NPU 资源。 |

#### 3. 资源耗尽与限流边界 (Resource Exhaustion & Admission Control)

边缘设备算力和内存严格受限，必须有明确的防雪崩机制。

| 边界场景 | 触发条件 | 系统表现与控制策略 |
| :--- | :--- | :--- |
| **NPU 算力/显存超售** | 用户配置任务超出了硬件理论并发上限。 | **准入控制 (Admission Control)**：Go 端启动任务前，校验当前已分配的算力权重/显存指标。超标时直接拒绝启动（返回资源不足错误），禁止强行启动导致全局 OOM 或帧率大幅跌落。 |
| **磁盘存储打满 (100%)** | 存储阈值清理异常，或短时间内爆发大量报警写入。 | 1. PG 数据库使用表分区管理，执行直接 `DROP TABLE` 提速清理；<br>2. ZLM 录像和抓拍文件写入报错时不影响系统主流程；<br>3. 系统预留 5% 缓冲区（磁盘达 95% 视为只读，拒绝新图片存入），并产生“系统存储空间不足”最高优先级事件。 |
| **内存泄漏边界 (OOM)** | C++ 处理高分辨率图像或视频解码帧（AVFrame）未及时释放。 | 1. C++ 引擎启动时设置 `cgroups` 内存软限；<br>2. 监控系统若发现内存持续飙升至阈值（如 90%），生成告警；<br>3. 极端情况下 OOM Killer 杀掉 C++ 进程，转入**进程崩溃恢复**流程。 |

#### 4. 网络恶劣环境异常 (Adverse Network Conditions)

| 边界场景 | 触发条件 | 系统表现与控制策略 |
| :--- | :--- | :--- |
| **视频流极度丢包/花屏** | IPC 到边缘设备的网线老化或无线网络丢包率高。 | 1. 硬件解码器可能解出不完整帧（绿屏/灰块）；<br>2. C++ 端应检测解码帧异常标志（如 `AV_FRAME_FLAG_CORRUPT`），对于损坏帧直接丢弃，**绝对禁止送入模型推理**，防止提取错误特征或崩溃。 |
| **信令风暴/重连雪崩** | 核心交换机重启，几十路摄像头同时掉线后同时重连。 | 1. Go 端任务重连必须引入**随机抖动（Jitter）和指数退避（Exponential Backoff）**；<br>2. 防止瞬间并发拉流将 ZLM 或网络带宽打爆。 |
| **告警 Webhook 平台失联** | 客户业务后端宕机或网络隔离。 | 系统产生的告警事件堆积：<br>1. 引入内存或轻量本地持久化队列存储事件；<br>2. 重试达到最大次数后，将告警状态标记为失败入库；<br>3. 队列达到容量上限时，采用**丢弃最旧数据（Ring Buffer）**策略，防止挤爆边缘盒子内存。 |

#### 5. 时序与竞态边界 (Timing & Race Conditions)

| 边界场景 | 触发条件 | 系统表现与控制策略 |
| :--- | :--- | :--- |
| **热更新与任务停止竞态** | 运维人员点击“更新算法包”的瞬间，另一用户点击“停止”该算法对应的任务。 | 1. 算法管理器严格使用**引用计数（Ref-counting）+ 读写锁**管理 `.so`；<br>2. 任务请求停止时引用计数 -1，为 0 时执行卸载释放；<br>3. 新任务优先绑定新的句柄，异步卸载不污染内存。 |
| **前端过载 (Browser OOM)** | 浏览器开启 9 宫格预览，且每秒上百个告警 BBox 推送。 | 1. 前端 WebSocket 接收 JSON 结果时应有帧率限制（如最大 15FPS 渲染）；<br>2. 队列积压时主动丢帧，只渲染最新帧；<br>3. Canvas 绘制必须使用 `requestAnimationFrame`，防止标签堆叠导致内存泄漏。 |

---
