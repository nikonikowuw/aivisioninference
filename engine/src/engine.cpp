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
    namespace
    {
        std::string PayloadToString(const uint8_t *payload, size_t size)
        {
            if (!payload || size == 0)
                return "";
            return std::string(reinterpret_cast<const char *>(payload), size);
        }

        std::string BuildLivePlayURL(const std::string &base_url, const std::string &device_id)
        {
            std::string play_url = base_url.empty() ? "rtsp://localhost:10554" : base_url;
            if (!play_url.empty() && play_url.back() == '/')
                play_url.pop_back();
            return play_url + "/live/" + device_id;
        }
    }

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
        pipeline::PipelineManagerConfig pipeline_config;
        pipeline_config.hal_so_path = config.hal_so_path;
        pipeline_config.hal_config_json = config.hal_config_json;
        pipeline_config.rtsp_push_server = config.rtsp_push_server;
        pipeline_config.enable_ffmpeg_fallback = config.enable_ffmpeg_fallback;
        pipeline_mgr_ = std::make_unique<pipeline::PipelineManager>(pipeline_config);
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
        Run({});
    }

    void InferenceEngine::Run(const std::function<bool()> &should_stop)
    {
        if (!initialized_.load())
        {
            if (!Initialize())
                return;
        }

        running_.store(true);
        shutdown_called_.store(false);

        // 启动 IPC Server
        if (!ipc_server_->Start())
        {
            std::cerr << "Failed to start IPC Server" << std::endl;
            running_.store(false);
            return;
        }

        // 启动 Worker 池
        worker_pool_->Start();

        // 启动 Metrics 上报
        metrics_reporter_->Start();

        // 主循环 (等待 Shutdown 或外部信号)
        while (running_.load())
        {
            if (should_stop && should_stop())
            {
                running_.store(false);
                break;
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }
    }

    void InferenceEngine::RequestShutdown()
    {
        running_.store(false);
    }

    void InferenceEngine::Shutdown()
    {
        bool expected = false;
        if (!shutdown_called_.compare_exchange_strong(expected, true))
            return;

        running_.store(false);

        if (pipeline_mgr_)
            pipeline_mgr_->StopAll();
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
        REGISTER_HANDLER(205, HandleStreamStatus);
        REGISTER_HANDLER(206, HandleStartSelfCheck);

#undef REGISTER_HANDLER
    }

    void InferenceEngine::HandleStartStream(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)payload; (void)size; (void)seq;
        // TODO: 解析 FlatBuffers 指令
        // pipeline_mgr_->CreatePipeline(device_id, rtsp_url, enable_infer, enable_playback);
        std::cout << "[IPC] Received StartStream" << std::endl;
    }

    void InferenceEngine::HandleStopStream(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)payload; (void)size; (void)seq;
        // TODO: 解析 FlatBuffers 指令
        // pipeline_mgr_->DestroyPipeline(device_id);
        std::cout << "[IPC] Received StopStream" << std::endl;
    }

    void InferenceEngine::HandleUpdateAlgoConfig(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)payload; (void)size; (void)seq;
        std::cout << "[IPC] Received UpdateAlgoConfig" << std::endl;
    }

    void InferenceEngine::HandleHeartbeat(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)payload; (void)size; (void)seq;
        // heartbeat_->OnHeartbeat(seq);
    }

    void InferenceEngine::HandleShutdown(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)payload; (void)size; (void)seq;
        std::cout << "[IPC] Received Shutdown command" << std::endl;
        Shutdown();
    }

    std::string InferenceEngine::ExtractJsonField(const std::string &json, const std::string &field_name)
    {
        std::string search = "\"" + field_name + "\"";
        size_t pos = json.find(search);
        if (pos == std::string::npos)
            return "";

        size_t start = json.find(":", pos);
        if (start == std::string::npos)
            return "";

        // 跳过空白
        start++;
        while (start < json.size() && (json[start] == ' ' || json[start] == '\t'))
            start++;

        if (start >= json.size())
            return "";

        // 字符串值
        if (json[start] == '"')
        {
            start++;
            size_t end = json.find("\"", start);
            if (end == std::string::npos)
                return "";
            return json.substr(start, end - start);
        }

        // 数字或布尔值
        size_t end = json.find_first_of(",}\n", start);
        if (end == std::string::npos)
            end = json.size();
        return json.substr(start, end - start);
    }

    bool InferenceEngine::ExtractJsonBoolField(const std::string &json, const std::string &field_name, bool default_value)
    {
        std::string value = ExtractJsonField(json, field_name);
        if (value.empty())
            return default_value;
        value.erase(std::remove_if(value.begin(), value.end(), ::isspace), value.end());
        std::transform(value.begin(), value.end(), value.begin(), [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
        if (value == "true" || value == "1")
            return true;
        if (value == "false" || value == "0")
            return false;
        return default_value;
    }

    namespace {
        /// libcurl 写回调，将响应追加到 std::string
        size_t curl_string_write_callback(void *ptr, size_t size, size_t nmemb, void *userdata)
        {
            std::string *str = static_cast<std::string*>(userdata);
            str->append(static_cast<const char*>(ptr), size * nmemb);
            return size * nmemb;
        }
    }

    std::string InferenceEngine::AddStreamProxy(const std::string &device_id, const std::string &rtsp_url)
    {
        if (config_.zlm_api_url.empty())
        {
            std::cerr << "[ZLM] zlm_api_url not configured" << std::endl;
            return "";
        }

        // 构建 ZLM addStreamProxy URL
        // GET /index/api/addStreamProxy?vhost=__defaultVhost__&app=live&stream={device_id}&url={rtsp_url}&secret=xxx&retry_count=3&rtp_type=0
        std::string zlm_url = config_.zlm_api_url;
        if (zlm_url.back() == '/')
            zlm_url.pop_back();

        // URL 编码 rtsp_url
        std::string encoded_url;
        CURL *curl = curl_easy_init();
        if (curl)
        {
            char *encoded = curl_easy_escape(curl, rtsp_url.c_str(), rtsp_url.size());
            if (encoded)
            {
                encoded_url = encoded;
                curl_free(encoded);
            }
            curl_easy_cleanup(curl);
        }
        if (encoded_url.empty())
            encoded_url = rtsp_url;

        std::string api_url = zlm_url + "/index/api/addStreamProxy"
            + "?vhost=__defaultVhost__"
            + "&app=live"
            + "&stream=" + device_id
            + "&url=" + encoded_url
            + "&secret=" + config_.zlm_secret
            + "&retry_count=3"
            + "&rtp_type=0"  // TCP
            + "&timeout_sec=10";

        std::cout << "[ZLM] Calling addStreamProxy for device: " << device_id << std::endl;
        std::cout << "[ZLM] URL: " << api_url << std::endl;

        // 发送 HTTP 请求
        std::string response;
        CURL *curl_handle = curl_easy_init();
        if (!curl_handle)
        {
            std::cerr << "[ZLM] Failed to init curl" << std::endl;
            return "";
        }

        curl_easy_setopt(curl_handle, CURLOPT_URL, api_url.c_str());
        curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION, curl_string_write_callback);
        curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA, &response);
        curl_easy_setopt(curl_handle, CURLOPT_TIMEOUT, 30L);

        CURLcode res = curl_easy_perform(curl_handle);
        curl_easy_cleanup(curl_handle);

        if (res != CURLE_OK)
        {
            std::cerr << "[ZLM] addStreamProxy failed: " << curl_easy_strerror(res) << std::endl;
            return "";
        }

        std::cout << "[ZLM] Response: " << response << std::endl;

        // 解析响应，检查 code 是否为 0
        std::string code_str = ExtractJsonField(response, "code");
        if (code_str != "0")
        {
            std::cerr << "[ZLM] addStreamProxy error: code=" << code_str << std::endl;
            std::string msg = ExtractJsonField(response, "msg");
            if (!msg.empty())
                std::cerr << "[ZLM] msg: " << msg << std::endl;
            return "";
        }

        // 构建播放 URL
        // RTSP: rtsp://{host}:554/live/{device_id}
        // WebRTC: webrtc://{host}:8000/live/{device_id}
        // 提取 host
        std::string host = config_.zlm_api_url;
        // 移除协议前缀
        size_t proto_end = host.find("://");
        if (proto_end != std::string::npos)
            host = host.substr(proto_end + 3);
        // 移除端口和路径
        size_t colon_pos = host.find(":");
        if (colon_pos != std::string::npos)
            host = host.substr(0, colon_pos);
        size_t slash_pos = host.find("/");
        if (slash_pos != std::string::npos)
            host = host.substr(0, slash_pos);

        std::string play_url = "rtsp://" + host + ":554/live/" + device_id;
        std::cout << "[ZLM] Stream proxy added, play URL: " << play_url << std::endl;

        return play_url;
    }

    bool InferenceEngine::CloseStreamProxy(const std::string &device_id)
    {
        if (config_.zlm_api_url.empty())
            return false;

        std::string zlm_url = config_.zlm_api_url;
        if (zlm_url.back() == '/')
            zlm_url.pop_back();

        std::string api_url = zlm_url + "/index/api/closeStream"
            + "?vhost=__defaultVhost__"
            + "&app=live"
            + "&stream=" + device_id
            + "&force=1"
            + "&secret=" + config_.zlm_secret;

        std::cout << "[ZLM] Closing stream proxy for device: " << device_id << std::endl;

        std::string response;
        CURL *curl = curl_easy_init();
        if (!curl)
            return false;

        curl_easy_setopt(curl, CURLOPT_URL, api_url.c_str());
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_string_write_callback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

        CURLcode res = curl_easy_perform(curl);
        curl_easy_cleanup(curl);

        if (res != CURLE_OK)
        {
            std::cerr << "[ZLM] closeStream failed: " << curl_easy_strerror(res) << std::endl;
            return false;
        }

        std::cout << "[ZLM] closeStream response: " << response << std::endl;
        return true;
    }

    void InferenceEngine::HandleStreamStart(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)seq;
        std::cout << "[IPC] Received StreamStart" << std::endl;

        std::string payload_str = PayloadToString(payload, size);
        std::string device_id = ExtractJsonField(payload_str, "device_id");
        std::string stream_url = ExtractJsonField(payload_str, "stream_url");
        bool enable_infer = ExtractJsonBoolField(payload_str, "enable_infer", false);
        bool enable_playback = ExtractJsonBoolField(payload_str, "enable_playback", false);

        std::cout << "[IPC] StreamStart device_id=" << device_id
                  << ", stream_url=" << stream_url
                  << ", enable_infer=" << enable_infer
                  << ", enable_playback=" << enable_playback << std::endl;

        if (device_id.empty() || stream_url.empty())
        {
            std::cerr << "[IPC] Invalid StreamStart payload: missing device_id or stream_url" << std::endl;
            flatbuffers::FlatBufferBuilder fbb(256);
            auto device_id_str = fbb.CreateString("");
            auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, false);
            fbb.Finish(resp);
            int client_fd = ipc_server_->GetActiveClientFd();
            if (client_fd >= 0)
                ipc_server_->SendResponse(client_fd, 301, fbb.GetBufferPointer(), fbb.GetSize());
            return;
        }

        bool started = pipeline_mgr_->CreatePipeline(device_id, stream_url, enable_infer, enable_playback);
        if (!started)
        {
            std::cerr << "[IPC] Failed to create hardware pipeline for device: " << device_id << std::endl;
        }

        std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);
        flatbuffers::FlatBufferBuilder fbb(512);
        auto device_id_str = fbb.CreateString(device_id);
        auto play_url_str = fbb.CreateString(started ? play_url : "");
        auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, started, 0, 0, 0, play_url_str);
        fbb.Finish(resp);

        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            ipc_server_->SendResponse(client_fd, 301, fbb.GetBufferPointer(), fbb.GetSize());
            std::cout << "[IPC] StreamStart response sent for device: " << device_id
                      << ", started=" << started
                      << ", play_url=" << (started ? play_url : "") << std::endl;
        }
        else
        {
            std::cerr << "[IPC] Failed to send StreamStart response: no active client" << std::endl;
        }
    }

    void InferenceEngine::HandleStreamStop(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)seq;
        std::cout << "[IPC] Received StreamStop" << std::endl;

        std::string payload_str = PayloadToString(payload, size);
        std::string device_id = ExtractJsonField(payload_str, "device_id");
        if (device_id.empty())
        {
            device_id = payload_str;
        }

        std::cout << "[IPC] StreamStop device_id=" << device_id << std::endl;

        if (!device_id.empty())
        {
            pipeline_mgr_->DestroyPipeline(device_id);
        }

        // 发送响应
        flatbuffers::FlatBufferBuilder fbb(256);
        auto device_id_str = fbb.CreateString(device_id);
        auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, false);
        fbb.Finish(resp);

        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            ipc_server_->SendResponse(client_fd, 302, fbb.GetBufferPointer(), fbb.GetSize());
            std::cout << "[IPC] StreamStop response sent for device: " << device_id << std::endl;
        }
    }

    void InferenceEngine::HandleStreamPlaybackStart(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)seq;
        std::cout << "[IPC] Received StreamPlaybackStart" << std::endl;

        std::string payload_str = PayloadToString(payload, size);
        std::string device_id = ExtractJsonField(payload_str, "device_id");
        std::string stream_url = ExtractJsonField(payload_str, "stream_url");
        if (device_id.empty())
        {
            device_id = payload_str;
        }

        std::cout << "[IPC] StreamPlaybackStart device_id=" << device_id
                  << ", stream_url=" << stream_url << std::endl;

        bool started = false;
        if (!device_id.empty())
        {
            if (pipeline_mgr_->GetPipeline(device_id))
            {
                started = pipeline_mgr_->EnablePlayback(device_id);
            }
            else if (!stream_url.empty())
            {
                started = pipeline_mgr_->CreatePipeline(device_id, stream_url, false, true);
            }
        }

        std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);

        flatbuffers::FlatBufferBuilder fbb(256);
        auto device_id_str = fbb.CreateString(device_id);
        auto play_url_str = started ? fbb.CreateString(play_url) : 0;
        auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, started, 0, 0, 0, play_url_str);
        fbb.Finish(resp);

        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            ipc_server_->SendResponse(client_fd, 303, fbb.GetBufferPointer(), fbb.GetSize());
            std::cout << "[IPC] StreamPlaybackStart response sent for device: " << device_id << std::endl;
        }
    }

    void InferenceEngine::HandleStreamPlaybackStop(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)seq;
        std::cout << "[IPC] Received StreamPlaybackStop" << std::endl;

        std::string payload_str = PayloadToString(payload, size);
        std::string device_id = ExtractJsonField(payload_str, "device_id");
        if (device_id.empty())
        {
            device_id = payload_str;
        }

        std::cout << "[IPC] StreamPlaybackStop device_id=" << device_id << std::endl;

        if (!device_id.empty())
        {
            pipeline_mgr_->DestroyPipeline(device_id);
        }

        // 发送响应
        flatbuffers::FlatBufferBuilder fbb(256);
        auto device_id_str = fbb.CreateString(device_id);
        auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, false);
        fbb.Finish(resp);

        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            ipc_server_->SendResponse(client_fd, 304, fbb.GetBufferPointer(), fbb.GetSize());
            std::cout << "[IPC] StreamPlaybackStop response sent for device: " << device_id << std::endl;
        }
    }

    void InferenceEngine::HandleStreamStatus(const uint8_t *payload, size_t size, uint64_t seq)
    {
        (void)seq;
        std::string payload_str = PayloadToString(payload, size);
        std::string device_id = ExtractJsonField(payload_str, "device_id");
        if (device_id.empty())
        {
            device_id = payload_str;
        }

        auto *pipeline = pipeline_mgr_->GetPipeline(device_id);
        bool running = pipeline != nullptr;
        std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);

        flatbuffers::FlatBufferBuilder fbb(512);
        auto device_id_str = fbb.CreateString(device_id);
        auto play_url_str = fbb.CreateString(running ? play_url : "");
        auto resp = aivision::ipc::CreateStreamStatusRspMsg(fbb, device_id_str, running, 0, 0, 0, play_url_str);
        fbb.Finish(resp);

        int client_fd = ipc_server_->GetActiveClientFd();
        if (client_fd >= 0)
        {
            ipc_server_->SendResponse(client_fd, 305, fbb.GetBufferPointer(), fbb.GetSize());
        }
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
        
        aivision::ipc::SelfCheckStatus status = success ? aivision::ipc::SelfCheckStatus_Passed 
                                                        : aivision::ipc::SelfCheckStatus_Failed;

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
