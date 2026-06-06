#include "engine.h"
#include "pipeline/pipeline_manager.h"
#include <iostream>

namespace aivision
{

    InferenceEngine::InferenceEngine(const EngineConfig &config)
        : config_(config)
    {
        // 创建组件
        ipc_server_ = std::make_unique<ipc::IPCServer>(ipc::IPCServerConfig{config.ipc_socket_path});
        heartbeat_ = std::make_unique<ipc::HeartbeatManager>();
        heartbeat_->SetServer(ipc_server_.get());

        // 原有的 StreamQueueManager 可能会被 PipelineManager 取代
        // 但为了兼容现有代码（如果有的话），先保留或重构
        queue_mgr_ = std::make_unique<pipeline::StreamQueueManager>();
        snapshot_mgr_ = std::make_unique<pipeline::SnapshotManager>();

        buffer_pool_ = std::make_unique<pipeline::HwBufferPool>(config.hw_buffer_pool_size);
        worker_pool_ = std::make_unique<pipeline::WorkerPool>(pipeline::WorkerPoolConfig{config.worker_count});

        hal_mgr_ = std::make_unique<pipeline::HALManager>();
        pipeline_mgr_ = std::make_unique<pipeline::PipelineManager>();
        algo_mgr_ = std::make_unique<algo::AlgoManager>();

        metrics_reporter_ = std::make_unique<monitor::MetricsReporter>(
            ipc_server_.get(), worker_pool_.get(), buffer_pool_.get(),
            queue_mgr_.get(), algo_mgr_.get(),
            monitor::MetricsReporterConfig{config.metrics_interval_ms});
    }

    InferenceEngine::~InferenceEngine()
    {
        Shutdown();
    }

    bool InferenceEngine::Initialize()
    {
        if (initialized_.load())
            return true;

        // 1. 初始化算法管理器
        // algo_mgr_->Initialize();

        // 2. 初始化 Worker 池
        worker_pool_->SetQueueManager(queue_mgr_.get());
        worker_pool_->SetSnapshotManager(snapshot_mgr_.get());
        worker_pool_->SetAlgoManager(algo_mgr_.get());

        // 3. 注册 IPC 指令处理器
        RegisterIPCCommandHandlers();

        initialized_.store(true);
        return true;
    }

    void InferenceEngine::Run()
    {
        if (!initialized_.load())
        {
            if (!Initialize())
                return;
        }

        running_.store(true);

        // 启动 IPC Server
        if (!ipc_server_->Start())
        {
            std::cerr << "Failed to start IPC Server" << std::endl;
            return;
        }

        // 启动 Worker 池
        worker_pool_->Start();

        // 启动 Metrics 上报
        metrics_reporter_->Start();

        // 主循环 (等待 Shutdown 或外部信号)
        while (running_.load())
        {
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }
    }

    void InferenceEngine::RequestShutdown()
    {
        running_.store(false);
    }

    void InferenceEngine::Shutdown()
    {
        if (!running_.load())
            return;
        running_.store(false);

        if (metrics_reporter_)
            metrics_reporter_->Stop();
        if (worker_pool_)
            worker_pool_->Stop();
        if (ipc_server_)
            ipc_server_->Stop();

        std::cout << "Engine shutdown complete" << std::endl;
    }

    void InferenceEngine::RegisterIPCCommandHandlers()
    {
#define REGISTER_HANDLER(code, method) \
    ipc_server_->RegisterHandler(code, [this](const uint8_t *p, size_t s, uint64_t seq) { method(p, s, seq); })

        REGISTER_HANDLER(101, HandleStartStream);
        REGISTER_HANDLER(102, HandleStopStream);
        REGISTER_HANDLER(103, HandleUpdateAlgoConfig);
        REGISTER_HANDLER(104, HandleHeartbeat);
        REGISTER_HANDLER(105, HandleShutdown);
        REGISTER_HANDLER(201, HandleStreamStart);
        REGISTER_HANDLER(202, HandleStreamStop);
        REGISTER_HANDLER(203, HandleStreamPlaybackStart);
        REGISTER_HANDLER(204, HandleStreamPlaybackStop);

#undef REGISTER_HANDLER
    }

    void InferenceEngine::HandleStartStream(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers 指令
        // pipeline_mgr_->CreatePipeline(device_id, rtsp_url, enable_infer, enable_playback);
        std::cout << "[IPC] Received StartStream" << std::endl;
    }

    void InferenceEngine::HandleStopStream(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers 指令
        // pipeline_mgr_->DestroyPipeline(device_id);
        std::cout << "[IPC] Received StopStream" << std::endl;
    }

    void InferenceEngine::HandleUpdateAlgoConfig(const uint8_t *payload, size_t size, uint64_t seq)
    {
        std::cout << "[IPC] Received UpdateAlgoConfig" << std::endl;
    }

    void InferenceEngine::HandleHeartbeat(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // heartbeat_->OnHeartbeat(seq);
    }

    void InferenceEngine::HandleShutdown(const uint8_t *payload, size_t size, uint64_t seq)
    {
        std::cout << "[IPC] Received Shutdown command" << std::endl;
        Shutdown();
    }

    void InferenceEngine::HandleStreamStart(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers StreamStartCmd
        // pipeline_mgr_->CreatePipeline(device_id, stream_url, enable_infer, enable_playback);
        std::cout << "[IPC] Received StreamStart" << std::endl;
    }

    void InferenceEngine::HandleStreamStop(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers StreamStopCmd
        // pipeline_mgr_->DestroyPipeline(device_id);
        std::cout << "[IPC] Received StreamStop" << std::endl;
    }

    void InferenceEngine::HandleStreamPlaybackStart(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers StreamPlaybackStartCmd
        // pipeline_mgr_->EnablePlayback(device_id);
        std::cout << "[IPC] Received StreamPlaybackStart" << std::endl;
    }

    void InferenceEngine::HandleStreamPlaybackStop(const uint8_t *payload, size_t size, uint64_t seq)
    {
        // TODO: 解析 FlatBuffers StreamPlaybackStopCmd
        // pipeline_mgr_->DisablePlayback(device_id);
        std::cout << "[IPC] Received StreamPlaybackStop" << std::endl;
    }

} // namespace aivision
