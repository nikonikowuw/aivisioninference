#ifndef AIVISION_ENGINE_H
#define AIVISION_ENGINE_H

// 推理引擎主入口 — 组装所有组件并启动。
// 职责：
//   1. 加载配置。
//   2. 创建 IPC Server 等待 Go 控制面连接。
//   3. 初始化硬件流水线 (HAL)。
//   4. 启动 Worker 线程池。
//   5. 启动 MetricsReporter。
//   6. 处理 Go 控制面指令 (启动/停止流、热更新等)。

#include <atomic>
#include <memory>
#include <string>

#include "ipc/ipc_server.h"
#include "ipc/heartbeat.h"
#include "pipeline/hw_buffer.h"
#include "pipeline/ring_queue.h"
#include "pipeline/snapshot.h"
#include "pipeline/worker_pool.h"
#include "pipeline/pipeline_manager.h"
#include "monitor/metrics_reporter.h"

namespace aivision
{

    /// 引擎配置
    struct EngineConfig
    {
        /// IPC UDS Socket 路径
        std::string ipc_socket_path = "/tmp/aivision_ipc.sock";

        /// Worker 线程数
        uint32_t worker_count = 4;

        /// 硬件缓冲池大小 (字节)
        size_t hw_buffer_pool_size = 256 * 1024 * 1024; // 256MB

        /// HAL 平台动态库路径 (可选，空字符串表示使用默认)
        std::string hal_so_path;

        /// HAL 配置 (JSON)
        std::string hal_config_json = "{}";

        /// Metrics 上报间隔 (毫秒)
        uint32_t metrics_interval_ms = 5000;

        /// 编码器配置 (JSON)
        std::string encoder_config_json =
            R"({"codec":"h264", "bitrate":4000000, "fps":25, "gop":50})";

        /// RTSP 推流目标服务器地址 (如 "rtsp://zlm:554")
        std::string rtsp_push_server = "rtsp://zlm:554";
    };

    /// 推理引擎主类
    class InferenceEngine
    {
    public:
        explicit InferenceEngine(const EngineConfig &config);
        ~InferenceEngine();

        /// 初始化引擎 (加载配置、创建组件)
        bool Initialize();

        /// 启动引擎 (阻塞，直到收到 Shutdown 信号)
        void Run();

        /// 停止引擎
        void Shutdown();

        /// 请求停止 (async-signal-safe，可从信号处理器调用)
        void RequestShutdown();

        /// 获取引擎是否运行中
        bool IsRunning() const { return running_.load(); }

        /// 获取引擎版本
        static std::string Version() { return "1.0.0"; }

        // ============================================================
        // 组件访问器
        // ============================================================

        ipc::IPCServer *GetIPCServer() { return ipc_server_.get(); }
        ipc::HeartbeatManager *GetHeartbeatManager() { return heartbeat_.get(); }
        pipeline::WorkerPool *GetWorkerPool() { return worker_pool_.get(); }
        pipeline::HwBufferPool *GetBufferPool() { return buffer_pool_.get(); }
        pipeline::StreamQueueManager *GetQueueManager() { return queue_mgr_.get(); }
        pipeline::SnapshotManager *GetSnapshotManager() { return snapshot_mgr_.get(); }
        pipeline::HALManager *GetHALManager() { return hal_mgr_.get(); }
        pipeline::PipelineManager *GetPipelineManager() { return pipeline_mgr_.get(); }
        algo::AlgoManager *GetAlgoManager() { return algo_mgr_.get(); }
        monitor::MetricsReporter *GetMetricsReporter() { return metrics_reporter_.get(); }

    private:
        /// 注册 IPC 指令处理器
        void RegisterIPCCommandHandlers();

        /// 处理 StartStream 指令
        void HandleStartStream(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StopStream 指令
        void HandleStopStream(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 UpdateAlgoConfig 指令
        void HandleUpdateAlgoConfig(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 Heartbeat 指令
        void HandleHeartbeat(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 Shutdown 指令
        void HandleShutdown(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StreamStart 指令
        void HandleStreamStart(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StreamStop 指令
        void HandleStreamStop(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StreamPlaybackStart 指令
        void HandleStreamPlaybackStart(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StreamPlaybackStop 指令
        void HandleStreamPlaybackStop(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StartSelfCheck 指令
        void HandleStartSelfCheck(const uint8_t *payload, size_t size, uint64_t seq);

        EngineConfig config_;
        std::atomic<bool> running_{false};
        std::atomic<bool> initialized_{false};

        // 组件
        std::unique_ptr<ipc::IPCServer> ipc_server_;
        std::unique_ptr<ipc::HeartbeatManager> heartbeat_;
        std::unique_ptr<pipeline::StreamQueueManager> queue_mgr_;
        std::unique_ptr<pipeline::SnapshotManager> snapshot_mgr_;
        std::unique_ptr<pipeline::HwBufferPool> buffer_pool_;
        std::unique_ptr<pipeline::WorkerPool> worker_pool_;
        std::unique_ptr<pipeline::HALManager> hal_mgr_;
        std::unique_ptr<pipeline::PipelineManager> pipeline_mgr_;
        std::unique_ptr<algo::AlgoManager> algo_mgr_;
        std::unique_ptr<monitor::MetricsReporter> metrics_reporter_;
    };

} // namespace aivision

#endif // AIVISION_ENGINE_H
