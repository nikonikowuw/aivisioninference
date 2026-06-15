# C++ Engine 与算法包开发规范

本规范适用于 `engine/`、`algorithms/`、`proto/flatbuf/` 相关代码。项目采用 Go 控制面 + C++ 数据面架构：Go 负责业务、任务、配置和结果落库，C++ Engine 负责视频流、推理管线、算法动态库加载和指标上报。

## 1. 工程边界

- `engine/` 是 C++17 推理引擎，可执行产物为 `aivision-engine`。
- `proto/flatbuf/` 定义 Go 与 C++ 之间的 FlatBuffers IPC 协议。
- `algorithms/<name>/<version>/` 是可上传、可自检、可动态加载的算法包。
- Go 后端不得直接承载视频解码和单帧推理重活；这些能力应放在 Engine 或算法包中。
- 算法包不得依赖 Go 进程内存或 Go 私有类型，只能通过约定 ABI、配置 JSON、FlatBuffers 消息和结果 JSON 交互。

## 2. Engine 架构约定

- 入口为 `engine/src/main.cpp`，核心编排在 `engine/src/engine.cpp`。
- IPC 服务在 `engine/src/ipc_server.cpp` 与 `engine/include/ipc/`。
- Pipeline 相关代码在 `engine/src/pipeline/` 与 `engine/include/pipeline/`。
- 算法动态库管理在 `engine/src/algo/` 与 `engine/include/algo/`。
- 指标上报在 `engine/src/monitor/`。
- 新增模块必须保持头文件在 `engine/include/`，实现文件在 `engine/src/`，并同步 `engine/CMakeLists.txt`。

## 3. C++ 编码规范

- 使用 C++17，禁止引入需要更高标准的语法或库能力。
- 资源管理优先使用 RAII、`std::unique_ptr`、`std::shared_ptr`、标准容器，避免裸 `new/delete`。
- 跨线程状态必须明确所有权和同步方式，使用 mutex、condition_variable、atomic 或无锁结构时必须说明生命周期。
- Engine 主流程禁止因单路流、单个算法或单条坏消息崩溃；错误应隔离到任务/流级别。
- 对外 IPC、动态库调用、线程入口必须捕获异常，禁止异常穿透进程边界或 C ABI 边界。
- 日志应可控、结构清晰；禁止在热路径中大量 `std::cout` / `std::cerr` 刷屏。

## 4. Pipeline 与性能

- 视频流处理链路应保持背压策略明确，队列容量、丢帧策略、线程数和停止流程必须可解释。
- 热路径避免不必要内存拷贝；硬件 buffer、DMA fd、编码帧和推理输入应尽量零拷贝传递。
- `detector_infer` 调用必须可超时、可失败、可统计耗时，不能无限阻塞 pipeline worker。
- 停止 stream 时必须释放解码器、encoder、算法实例、buffer、线程和 IPC 关联状态。
- 指标上报不能阻塞推理主路径，失败时应降级而非影响业务流。

## 5. FlatBuffers 协议规范

- Schema 文件位于 `proto/flatbuf/*.fbs`，`envelope.fbs` 是 IPC 信封入口。
- 字段演进必须遵守兼容性：新增可选字段优先，禁止重排字段 ID，废弃字段只标记 deprecated。
- 修改 schema 后必须同步生成 Go 和 C++ 代码，并更新 `proto/flatbuf/CHANGELOG.md` 与 `COMPATIBILITY.md`。
- Go 生成目标为 `app/internal/pkg/ipc/flatbuf`，C++ 生成目标为 `engine/include/ipc`。
- 不要手工修改生成的 FlatBuffers 代码。
- 消息必须包含可追踪的 sequence、timestamp、task/stream 标识，便于跨进程定位问题。

## 6. 算法包目录规范

每个算法包遵循：

```text
algorithms/<algorithm_name>/<version>/
├── algo_meta.yaml
├── label_map.json
├── models/
├── src/
├── CMakeLists.txt 或 build.sh
├── test.sh
└── README.md
```

- `algo_meta.yaml` 是平台解析入口，必须包含算法标识、版本、领域、能力、结果 schema、参数 schema。
- `label_map.json` 必须与推理输出中的 `category_code` 完全一致。
- 算法包应提供本地自检脚本，能验证动态库符号、模型加载和基础推理。
- 发布包应自包含，不能依赖部署现场额外安装未声明的系统库。

## 7. C ABI 契约

- 算法动态库通过 `engine/include/algo/abi_contract.h` 定义的 C ABI 与 Engine 交互。
- 导出函数必须使用 `extern "C"`，避免 C++ 名字改编。
- 初始化函数解析 JSON 配置并返回 opaque handle，失败返回空指针或约定错误。
- 推理函数接收硬件 buffer 描述、上下文 JSON、输出结构，并返回稳定错误码。
- 结果内存分配和释放必须成对约定，严禁跨模块使用不匹配 allocator。
- 销毁函数必须幂等释放模型、显存、线程、句柄和临时缓存。
- C ABI 外层必须捕获所有 C++ 异常，不能把异常抛给 Engine。

## 8. 结果 JSON 与配置同步

- 推理结果 JSON 应稳定、可版本化，字段命名与 Go DTO/数据库映射保持一致。
- `category_code` 必须在约定范围内，并与 `label_map.json` 对齐。
- bbox、confidence、track_id、timestamp、roi 等字段必须明确单位和坐标系。
- C++ 默认阈值、类别、输入尺寸、NMS 参数变化时，必须同步 `algo_meta.yaml` 的 `ai_params_schema`。
- 算法错误不要直接返回面向用户的中文句子；应返回错误码或英文标识，由 Go/前端负责 i18n 映射。

## 9. 构建与验证

- Engine 构建以 `engine/CMakeLists.txt` 和 `engine/Makefile` 为准。
- 常用命令：
  - `cd engine && make build`
  - `cd engine && make clean`
  - `cd engine && make install`
- 算法包验证优先运行包内 `build.sh`、`test.sh` 或 README 指定命令。
- 修改 FlatBuffers 后应执行对应 `flatc --go` 与 `flatc --cpp` 生成命令，并编译 Go 与 Engine 双端。
- 涉及 pipeline、ABI、IPC 的修改必须至少验证：启动、加载算法、开始流、停止流、异常算法包、进程退出清理。
