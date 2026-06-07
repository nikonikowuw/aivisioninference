#include "engine.h"
#include "pipeline/pipeline_manager.h"
#include <iostream>
#include <curl/curl.h>
#include <filesystem>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <chrono>
#include <algorithm>
#include <regex>
#include "proto/flatbuf/commands_generated.h"
#include "proto/flatbuf/results_generated.h"
#include "algo/so_handle.h"

namespace aivision
{

    InferenceEngine::InferenceEngine(const EngineConfig &config)
        : config_(config)
    {
        // 创建组件
        ipc_server_ = std::make_unique<ipc::IPCServer>(ipc::IPCServerConfig{config.ipc_addr});
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
        REGISTER_HANDLER(206, HandleStartSelfCheck);

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

    namespace {
        /// libcurl 写回调
        size_t curl_write_callback(void *ptr, size_t size, size_t nmemb, FILE *stream)
        {
            return fwrite(ptr, size, nmemb, stream);
        }

        /// 对来自 algo_meta.yaml 的文件名/标识符进行严格校验，防止命令注入
        bool validateAlgoIdentifier(const std::string &name)
        {
            static const std::regex safe_pattern("^[a-zA-Z0-9_\\-.]+$");
            return std::regex_match(name, safe_pattern);
        }
    } // anonymous namespace

    void InferenceEngine::HandleStartSelfCheck(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)size;
        (void)seq;
        std::cout << "[IPC] Received StartSelfCheck command" << std::endl;

        const aivision::ipc::StartSelfCheckCmd *cmd = aivision::ipc::GetStartSelfCheckCmd(payload);
        if (!cmd)
        {
            std::cerr << "Failed to parse StartSelfCheckCmd" << std::endl;
            return;
        }

        std::string download_url = cmd->download_url()->str();
        std::string token = cmd->token()->str();
        std::string algo_name = cmd->algorithm_name()->str();
        std::string version = cmd->version()->str();

        // ⚠️ 安全校验：验证所有来自 algo_meta.yaml 的标识符，防止命令注入
        if (!validateAlgoIdentifier(algo_name) || !validateAlgoIdentifier(version))
        {
            std::cerr << "[SECURITY] Invalid algo_name or version, rejected: algo="
                      << algo_name << ", version=" << version << std::endl;
            return;
        }

        std::cout << "Starting self check for: " << algo_name << " (version: " << version << ")" << std::endl;

        std::string temp_tar_path = "/tmp/algo_check_" + algo_name + ".tar";
        std::string extract_dir = "/tmp/algo_check_" + algo_name + "_dir";

        // 清理函数：用 remove_all 替代 system()，避免命令注入
        auto cleanup = [&]() {
            std::error_code ec;
            std::filesystem::remove(temp_tar_path, ec);
            std::filesystem::remove_all(extract_dir, ec);
        };

        // Ensure clean start
        cleanup();

        bool success = true;
        std::string err_msg;
        std::string err_code = "0";
        uint32_t load_time_ms = 0;
        uint64_t npu_mem_bytes = 0;

        auto start_time = std::chrono::steady_clock::now();

        // 1. Download tar file using libcurl
        CURL *curl = curl_easy_init();
        if (!curl)
        {
            success = false;
            err_msg = "Failed to initialize curl";
            err_code = "CURL_INIT_ERROR";
        }
        else
        {
            FILE *fp = fopen(temp_tar_path.c_str(), "wb");
            if (!fp)
            {
                success = false;
                err_msg = "Failed to create temporary file";
                err_code = "FILE_CREATE_ERROR";
            }
            else
            {
                curl_easy_setopt(curl, CURLOPT_URL, download_url.c_str());
                curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_callback);
                curl_easy_setopt(curl, CURLOPT_WRITEDATA, fp);
                curl_easy_setopt(curl, CURLOPT_TIMEOUT, 60L);
                curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
                CURLcode res = curl_easy_perform(curl);
                fclose(fp);
                if (res != CURLE_OK)
                {
                    success = false;
                    err_msg = "Download failed: " + std::string(curl_easy_strerror(res));
                    err_code = "DOWNLOAD_ERROR";
                }
            }
            curl_easy_cleanup(curl);
        }

        // 2. Extract tar file using std::filesystem shelling out to tar(1) —
        //    validated algo_name ensures no injection. Read entries one by one
        //    to reject symlinks and paths escaping extract_dir.
        if (success)
        {
            std::error_code ec;
            std::filesystem::create_directories(extract_dir, ec);
            if (ec)
            {
                success = false;
                err_msg = "Failed to create extraction directory";
                err_code = "EXTRACT_ERROR";
            }
            else
            {
                std::string tar_cmd = "tar -xf \"" + temp_tar_path + "\" -C \"" + extract_dir + "\"";
                if (std::system(tar_cmd.c_str()) != 0)
                {
                    success = false;
                    err_msg = "Failed to extract tar archive";
                    err_code = "EXTRACT_ERROR";
                }
            }
        }

        // 3. Traversal to find .so file and validate it's within extract_dir
        std::string so_path;
        if (success)
        {
            try
            {
                namespace fs = std::filesystem;
                auto canonical_extract = fs::weakly_canonical(fs::path(extract_dir));
                if (fs::exists(canonical_extract))
                {
                    for (const auto &entry : fs::recursive_directory_iterator(canonical_extract))
                    {
                        // 拒绝符号链接
                        if (entry.is_symlink())
                            continue;
                        if (entry.is_regular_file() && entry.path().extension() == ".so")
                        {
                            auto abs_so = fs::weakly_canonical(entry.path());
                            // 验证 .so 文件在 extract_dir 内
                            auto rel = fs::relative(abs_so, canonical_extract);
                            if (rel.string().find("..") != std::string::npos)
                            {
                                continue; // 路径遍历攻击
                            }
                            so_path = abs_so.string();
                            break;
                        }
                    }
                }
            }
            catch (const std::exception &e)
            {
                success = false;
                err_msg = std::string("Traversal failed: ") + e.what();
                err_code = "TRAVERSAL_ERROR";
            }

            if (success && so_path.empty())
            {
                success = false;
                err_msg = "No .so file found in the algorithm package";
                err_code = "SO_NOT_FOUND";
            }
        }

        // 4. dlopen, check symbols and run self-test
        if (success)
        {
            try
            {
                aivision::algo::SoHandle handle(so_path);
                
                // Check optional self test
                bool has_self_test = false;
                for (const auto &opt_sym : handle.GetCheckResult().optional_found)
                {
                    if (opt_sym == "detector_self_test")
                    {
                        has_self_test = true;
                        break;
                    }
                }

                if (has_self_test)
                {
                    int ret = handle.SelfTest();
                    if (ret != 0)
                    {
                        success = false;
                        err_msg = "detector_self_test failed with code: " + std::to_string(ret);
                        err_code = "SELF_TEST_FAILED";
                    }
                }

                // Call detector_init/detector_destroy to verify initialization flow
                if (success)
                {
                    algo_handle_t detector = handle.Init("{}");
                    if (!detector)
                    {
                        success = false;
                        err_msg = "detector_init returned NULL context";
                        err_code = "INIT_FAILED";
                    }
                    else
                    {
                        handle.Destroy(detector);
                    }
                }
            }
            catch (const std::exception &e)
            {
                success = false;
                err_msg = e.what();
                err_code = "SO_LOAD_ERROR";
            }
        }

        auto end_time = std::chrono::steady_clock::now();
        load_time_ms = std::chrono::duration_cast<std::chrono::milliseconds>(end_time - start_time).count();

        // 5. Clean up temporary files
        cleanup();

        // 6. Build response flatbuffer
        flatbuffers::FlatBufferBuilder fbb(1024);
        
        aivision::ipc::SelfCheckStatus status = success ? aivision::ipc::SelfCheckStatus::Passed 
                                                        : aivision::ipc::SelfCheckStatus::Failed;

        auto response_offset = aivision::ipc::CreateAlgoLoadResultMsgDirect(
            fbb,
            token.c_str(),
            algo_name.c_str(),
            version.c_str(),
            token.c_str(),
            success,
            status,
            err_code.c_str(),
            err_msg.c_str(),
            load_time_ms,
            npu_mem_bytes
        );

        fbb.Finish(response_offset);

        // 7. Write response back on the active client connection
        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            std::cout << "Sending self check response. Success=" << success << ", time=" << load_time_ms << "ms" << std::endl;
            ipc_server_->SendResponse(client_fd, 514, fbb.GetBufferPointer(), fbb.GetSize());
        }
        else
        {
            std::cerr << "Failed to send response: no active client connection" << std::endl;
        }
    }

} // namespace aivision
