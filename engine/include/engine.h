#ifndef AIVISION_ENGINE_H
#define AIVISION_ENGINE_H

// 推理引擎主入口 — 组装所有组件并启动。
// 职责：
//   1. 加载配置。
//   2. 启动 MQTT 控制面并接收 Go 控制面命令。
//   3. 初始化硬件流水线 (HAL)。
//   4. 启动 Worker 线程池。
//   5. 启动 MetricsReporter。
//   6. 通过 HTTP HeartbeatReporter 上报边缘节点状态。
//   7. 通过 MQTT 事件上报推理结果。

#include <atomic>
#include <flatbuffers/flatbuffers.h>
#include <functional>
#include <memory>
#include <string>

#include "pipeline/hw_buffer.h"
#include "pipeline/ring_queue.h"
#include "pipeline/snapshot.h"
#include "pipeline/worker_pool.h"
#include "pipeline/pipeline_manager.h"
#include "monitor/metrics_reporter.h"
#include "response_router.h"

namespace aivision
{

    /// 引擎配置
    struct EngineConfig
    {

        /// Worker 线程数
        uint32_t worker_count = 4;

        /// 硬件缓冲池大小 (字节)
        size_t hw_buffer_pool_size = 256 * 1024 * 1024; // 256MB

        /// HAL 平台动态库路径 (可选，空字符串表示使用默认)
        std::string hal_so_path;

        /// fallback HAL 平台动态库路径，必须同样能解码并输出帧给推理 Worker
        std::string fallback_hal_so_path;

        /// HAL 配置 (JSON)
        std::string hal_config_json = "{}";

        /// 是否允许无 HAL 或 HAL 启动失败时使用 FFmpeg 兜底转推
        bool enable_ffmpeg_fallback = true;

        /// Metrics 上报间隔 (毫秒)
        uint32_t metrics_interval_ms = 5000;

		/// Device-qualified concurrent preview capacity; zero means invalid/unconfigured.
		uint32_t max_preview_streams = 0;

        /// 编码器配置 (JSON)
        std::string encoder_config_json =
            R"({"codec":"h264", "bitrate":4000000, "fps":25, "gop":50})";

        /// RTSP 推流目标服务器地址 (如 "rtsp://localhost:10554")
        std::string rtsp_push_server = "rtsp://localhost:10554";

        /// ZLM API URL (如 "http://localhost:80")
        std::string zlm_api_url = "http://localhost:80";

        /// ZLM Secret
        std::string zlm_secret = "";

        /// 平台管理端 URL（用于心跳上报）
        std::string platform_url = "";

        /// 边缘节点 ID（平台注册后获取）
        std::string node_id = "";

        /// 引擎认证 Token（平台创建节点后获取）
        std::string auth_token = "";

        /// 算法包安装基目录
        std::string algo_dir = "/var/aivision/algo";

        /// 是否启用 MQTT NATIVE 客户端
        bool enable_mqtt = false;

        /// MQTT Broker 地址
        std::string mqtt_broker = "tcp://localhost:1883";

        /// MQTT 客户端 ID
        std::string mqtt_client_id = "";

        /// MQTT 用户名
        std::string mqtt_username = "";

        /// MQTT 密码
        std::string mqtt_password = "";

        /// 设备监控平台覆盖 (如 "rockchip", "nvidia", "ascend", "macos", "linux_generic" 等)
        std::string device_platform = "";

        /// 设备监控存储路径
        std::string device_storage_path = "/";

        /// 是否允许设备监控执行外部命令
        bool device_enable_external_commands = true;

        /// 设备监控外部命令超时时间 (毫秒)
        uint32_t device_command_timeout_ms = 1500;

        /// 设备监控轻量指标采样周期 (毫秒)
        uint32_t device_light_probe_interval_ms = 5000;

        /// 设备监控昂贵指标采样周期 (毫秒)
        uint32_t device_expensive_probe_interval_ms = 20000;
    };

    namespace monitor { class HeartbeatReporter; class DeviceMonitor; }
    class CommandDispatcher;
    class MqttControlPlane;

    /// 推理引擎主类
    class InferenceEngine
    {
        friend class CommandDispatcher;
    public:
        explicit InferenceEngine(const EngineConfig &config);
        ~InferenceEngine();

        /// 初始化引擎 (加载配置、创建组件)
        bool Initialize();

        /// 启动引擎 (阻塞，直到收到 Shutdown 信号)
        void Run();

        /// 启动引擎 (阻塞，直到收到 Shutdown 信号或外部退出条件)
        void Run(const std::function<bool()> &should_stop);

        /// 停止引擎
        void Shutdown();

        /// 请求停止 (async-signal-safe，可从信号处理器调用)
        void RequestShutdown();

        /// 获取引擎是否运行中
        bool IsRunning() const { return running_.load(); }

        /// 获取引擎版本
        static std::string Version() { return "1.0.0"; }

        /// 获取引擎配置引用
        const EngineConfig& GetConfig() const { return config_; }

        /// 获取当前 HAL 平台名称
        std::string GetHalPlatform() const;

        // ============================================================
        // 组件访问器
        // ============================================================

        ResponseRouter *GetResponseRouter() { return response_router_.get(); }
        pipeline::WorkerPool *GetWorkerPool() { return worker_pool_.get(); }
        pipeline::HwBufferPool *GetBufferPool() { return buffer_pool_.get(); }
        pipeline::StreamQueueManager *GetQueueManager() { return queue_mgr_.get(); }
        pipeline::SnapshotManager *GetSnapshotManager() { return snapshot_mgr_.get(); }
        pipeline::HALManager *GetHALManager() { return hal_mgr_.get(); }
        pipeline::PipelineManager *GetPipelineManager() { return pipeline_mgr_.get(); }
        algo::AlgoManager *GetAlgoManager() { return algo_mgr_.get(); }
        monitor::MetricsReporter *GetMetricsReporter() { return metrics_reporter_.get(); }
        monitor::DeviceMonitor *GetDeviceMonitor() { return device_monitor_.get(); }

        /// 统一发布 FlatBuffers 结果事件
        bool PublishEvent(uint16_t signal_type, flatbuffers::FlatBufferBuilder &fbb);

        CommandDispatcher *GetCommandDispatcher() { return command_dispatcher_.get(); }
        MqttControlPlane *GetMqttControlPlane() { return mqtt_control_plane_.get(); }

    private:
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

        /// 处理 StreamStatus 指令
        void HandleStreamStatus(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理 StartSelfCheck 指令
        void HandleStartSelfCheck(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理人脸库快照热更新指令
        void HandleFaceLibraryUpdate(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理算法按需预热指令
        void HandleAlgoWarmup(const uint8_t *payload, size_t size, uint64_t seq);

        /// 处理单张图片人脸特征提取指令
        void HandleFaceEmbeddingExtract(const uint8_t *payload, size_t size, uint64_t seq);

        /// 调用 ZLM addStreamProxy API 拉取 RTSP 流
        /// 返回 ZLM 的播放 URL，失败返回空字符串
        std::string AddStreamProxy(const std::string &device_id, const std::string &rtsp_url);

        /// 调用 ZLM closeStream API 停止拉流
        bool CloseStreamProxy(const std::string &device_id);

        EngineConfig config_;
        std::atomic<bool> running_{false};
        std::atomic<bool> initialized_{false};
        std::atomic<bool> shutdown_called_{false};

        // 组件
        std::unique_ptr<ResponseRouter> response_router_;
        std::unique_ptr<pipeline::StreamQueueManager> queue_mgr_;
        std::unique_ptr<pipeline::SnapshotManager> snapshot_mgr_;
        std::unique_ptr<pipeline::HwBufferPool> buffer_pool_;
        std::unique_ptr<pipeline::WorkerPool> worker_pool_;
        std::unique_ptr<pipeline::HALManager> hal_mgr_;
        std::unique_ptr<pipeline::PipelineManager> pipeline_mgr_;
        std::unique_ptr<algo::AlgoManager> algo_mgr_;
        std::unique_ptr<monitor::MetricsReporter> metrics_reporter_;

        // 心跳上报模块（向平台推送引擎状态）
        std::unique_ptr<monitor::HeartbeatReporter> heartbeat_reporter_;

        // 设备状态监控模块
        std::unique_ptr<monitor::DeviceMonitor> device_monitor_;

        // MQTT & Command Dispatcher
        std::unique_ptr<CommandDispatcher> command_dispatcher_;
        std::unique_ptr<MqttControlPlane> mqtt_control_plane_;
    };

} // namespace aivision

#endif // AIVISION_ENGINE_H
