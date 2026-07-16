#include "monitor/heartbeat_reporter.h"
#include "monitor/metrics_flattener.h"
#include "monitor/device_monitor.h"
#include "engine.h"
#include "algo/algo_manager.h"
#include "algo/algorithm_downloader.h"
#include "monitor/device_monitor.h"
#include "monitor/device_snapshot_json.h"
#include "pipeline/pipeline_manager.h"
#include <curl/curl.h>
#include <nlohmann/json.hpp>
#include <iostream>
#include <sstream>
#include <chrono>
#include <fstream>
#include <dirent.h>
#include <unistd.h>
#include <random>
#include <cmath>

#ifdef __APPLE__
#include <sys/types.h>
#include <sys/sysctl.h>
#include <ifaddrs.h>
#include <net/if.h>
#include <net/if_dl.h>
#else
#include <sys/sysinfo.h>
#endif

using json = nlohmann::json;

namespace aivision
{
    namespace monitor
    {
        namespace detail
        {
            // RAII wrapper for CURL resources to ensure cleanup on exception/early return.
            // Must outlive individual requests — used as a member of HeartbeatReporter
            // so the underlying connection pool and TLS sessions survive across heartbeats.
            class CurlHandle {
            private:
                CURL* curl_;
                struct curl_slist* headers_;

            public:
                CurlHandle() : curl_(curl_easy_init()), headers_(nullptr) {}

                ~CurlHandle() {
                    if (headers_) {
                        curl_slist_free_all(headers_);
                    }
                    if (curl_) {
                        curl_easy_cleanup(curl_);
                    }
                }

                // Delete copy constructor and assignment operator
                CurlHandle(const CurlHandle&) = delete;
                CurlHandle& operator=(const CurlHandle&) = delete;

                CURL* get() const { return curl_; }

                void add_header(const char* header) {
                    headers_ = curl_slist_append(headers_, header);
                }

                struct curl_slist* headers() const { return headers_; }

                bool is_valid() const { return curl_ != nullptr; }

                /// Reset all request-specific options while keeping the connection cache alive.
                /// Call before reusing this handle for a new request.
                void reset() {
                    if (headers_) {
                        curl_slist_free_all(headers_);
                        headers_ = nullptr;
                    }
                    if (curl_) {
                        curl_easy_reset(curl_);
                    }
                }
            };
        }

        namespace
        {
            uint64_t GetTotalMemory()
            {
                static const uint64_t cached_mem = []() {
#ifdef __APPLE__
                    int mib[2] = {CTL_HW, HW_MEMSIZE};
                    int64_t physical_memory = 0;
                    size_t length = sizeof(physical_memory);
                    if (sysctl(mib, 2, &physical_memory, &length, NULL, 0) == 0)
                    {
                        return static_cast<uint64_t>(physical_memory);
                    }
                    return 16ULL * 1024 * 1024 * 1024;
#else
                    std::ifstream file("/proc/meminfo");
                    std::string line;
                    if (file.is_open())
                    {
                        while (std::getline(file, line))
                        {
                            if (line.rfind("MemTotal:", 0) == 0)
                            {
                                std::stringstream ss(line);
                                std::string label;
                                uint64_t value = 0;
                                std::string unit;
                                ss >> label >> value >> unit;
                                if (unit == "kB") return value * 1024;
                                return value;
                            }
                        }
                    }
                    return 8ULL * 1024 * 1024 * 1024;
#endif
                }();
                return cached_mem;
            }

            std::string GetCPUModel()
            {
                static const std::string cached_model = []() {
#ifdef __APPLE__
                    char buffer[256];
                    size_t buffer_len = sizeof(buffer);
                    if (sysctlbyname("machdep.cpu.brand_string", &buffer, &buffer_len, NULL, 0) == 0)
                    {
                        return std::string(buffer);
                    }
                    return std::string("Apple Silicon");
#else
                    std::ifstream file("/proc/cpuinfo");
                    std::string line;
                    if (file.is_open())
                    {
                        while (std::getline(file, line))
                        {
                            if (line.rfind("model name", 0) == 0)
                            {
                                size_t colon = line.find(":");
                                if (colon != std::string::npos)
                                {
                                    std::string model = line.substr(colon + 1);
                                    // Trim leading/trailing spaces
                                    size_t start = model.find_first_not_of(" \t");
                                    size_t end = model.find_last_not_of(" \t");
                                    if (start != std::string::npos && end != std::string::npos)
                                    {
                                        return model.substr(start, end - start + 1);
                                    }
                                    return model;
                                }
                            }
                        }
                    }
                    return std::string("ARM Cortex-A55");
#endif
                }();
                return cached_model;
            }

            /// 从 /proc/loadavg 读取负载数据
            double ReadLoad1m()
            {
#ifdef __APPLE__
                // sysctl vm.loadavg returns fixed-point load averages with the
                // layout { uint32_t ldavg[3]; long fscale; }. The struct loadavg
                // is not available in macOS user-space headers, so define inline.
                struct { uint32_t ldavg[3]; long fscale; } load_info;
                size_t len = sizeof(load_info);
                if (sysctlbyname("vm.loadavg", &load_info, &len, NULL, 0) == 0 &&
                    load_info.fscale > 0)
                {
                    return static_cast<double>(load_info.ldavg[0]) / load_info.fscale;
                }
                return 0.0;
#else
                std::ifstream file("/proc/loadavg");
                if (file.is_open())
                {
                    double l1, l5, l15;
                    file >> l1 >> l5 >> l15;
                    return l1;
                }
                return 0.0;
#endif
            }

            /// 从 /proc/net/dev 读取网络字节计数
            struct NetStats {
                uint64_t rx_bytes = 0;
                uint64_t tx_bytes = 0;
            };
            NetStats ReadNetDev()
            {
                NetStats stats;
#ifdef __APPLE__
                struct ifaddrs *ifap, *ifa;
                if (getifaddrs(&ifap) != 0) return stats;
                for (ifa = ifap; ifa; ifa = ifa->ifa_next) {
                    if (!ifa->ifa_addr) continue;
                    if (ifa->ifa_addr->sa_family != AF_LINK) continue;
                    if (ifa->ifa_flags & IFF_LOOPBACK) continue;
                    auto *ifdata = reinterpret_cast<struct if_data *>(ifa->ifa_data);
                    if (!ifdata) continue;
                    stats.rx_bytes += ifdata->ifi_ibytes;
                    stats.tx_bytes += ifdata->ifi_obytes;
                }
                freeifaddrs(ifap);
                return stats;
#else
                std::ifstream file("/proc/net/dev");
                if (!file.is_open()) return stats;
                std::string line;
                // Skip header lines
                std::getline(file, line); // header
                std::getline(file, line); // header
                while (std::getline(file, line))
                {
                    // Parse: inter-name: rx_bytes ... tx_bytes
                    size_t colon = line.find(':');
                    if (colon == std::string::npos) continue;
                    std::string iface = line.substr(0, colon);
                    // Skip loopback
                    if (iface.find("lo") != std::string::npos) continue;
                    std::stringstream ss(line.substr(colon + 1));
                    uint64_t rx, tx;
                    ss >> rx;
                    // Skip 7 fields to get to tx_bytes
                    for (int i = 0; i < 7; i++) { uint64_t skip; ss >> skip; }
                    ss >> tx;
                    stats.rx_bytes += rx;
                    stats.tx_bytes += tx;
                }
                return stats;
#endif
            }

            /// 从 /proc/uptime 读取运行时长（秒）
            uint64_t ReadUptime()
            {
#ifdef __APPLE__
                struct timeval boottime;
                size_t len = sizeof(boottime);
                int mib[2] = {CTL_KERN, KERN_BOOTTIME};
                if (sysctl(mib, 2, &boottime, &len, NULL, 0) == 0)
                {
                    time_t now;
                    time(&now);
                    return static_cast<uint64_t>(now - boottime.tv_sec);
                }
                return 0;
#else
                std::ifstream file("/proc/uptime");
                if (file.is_open())
                {
                    double up_secs;
                    file >> up_secs;
                    return static_cast<uint64_t>(up_secs);
                }
                return 0;
#endif
            }

            /// 从 /sys/class/thermal/ 读取核心温度（摄氏度）
            double ReadTemperature()
            {
#ifdef __APPLE__
                return 0.0;
#else
                std::ifstream file("/sys/class/thermal/thermal_zone0/temp");
                if (file.is_open())
                {
                    int milli_celsius = 0;
                    file >> milli_celsius;
                    return static_cast<double>(milli_celsius) / 1000.0;
                }
                return 0.0;
#endif
            }

            /// 统计系统进程和线程数
            struct ProcCounts {
                int processes = 0;
                int threads = 0;
            };
            ProcCounts CountProcesses()
            {
                ProcCounts counts;
#ifdef __APPLE__
                return counts;
#else
                DIR* dir = opendir("/proc");
                if (!dir) return counts;
                struct dirent* entry;
                while ((entry = readdir(dir)) != nullptr)
                {
                    // Process directories are numeric
                    char* end;
                    long pid = strtol(entry->d_name, &end, 10);
                    if (*end != '\0' || pid <= 0) continue;
                    counts.processes++;
                    // Count threads from /proc/[pid]/status
                    std::string status_path = "/proc/" + std::string(entry->d_name) + "/status";
                    std::ifstream sf(status_path);
                    std::string line;
                    while (std::getline(sf, line))
                    {
                        if (line.rfind("Threads:", 0) == 0)
                        {
                            std::stringstream ss(line);
                            std::string label;
                            int threads = 0;
                            ss >> label >> threads;
                            counts.threads += threads;
                            break;
                        }
                    }
                }
                closedir(dir);
                return counts;
#endif
            }
        }

        HeartbeatReporter::HeartbeatReporter(InferenceEngine* engine)
            : engine_(engine)
            , metrics_flattener_(std::make_unique<MetricsFlattener>())
            , last_flatten_timestamp_ms_(0)
        {
        }

        HeartbeatReporter::~HeartbeatReporter()
        {
            Stop();
        }

        void HeartbeatReporter::Start()
        {
            if (running_.load())
                return;

            running_.store(true);
            start_time_ = std::chrono::steady_clock::now();
            thread_ = std::make_unique<std::thread>(&HeartbeatReporter::ReportLoop, this);
        }

        void HeartbeatReporter::Stop()
        {
            if (!running_.load())
                return;

            running_.store(false);
            stop_cv_.notify_all();
            if (thread_ && thread_->joinable())
            {
                thread_->join();
            }
            thread_.reset();
        }

        void HeartbeatReporter::ReportLoop()
        {
            std::random_device rd;
            std::mt19937 gen(rd());
            std::uniform_int_distribution<int> jitter_dist(0, 1000); // 0-1000 ms jitter

            while (running_.load())
            {
                SendHeartbeat();

                // Compute sleep duration with exponential backoff and jitter
                int failures = consecutive_failures_;
                double base_seconds = 5.0;
                if (failures > 0)
                {
                    double backoff = base_seconds * std::pow(2.0, failures - 1);
                    if (backoff > 60.0)
                    {
                        backoff = 60.0;
                    }
                    base_seconds = backoff;
                }

                int jitter_ms = jitter_dist(gen);
                auto sleep_duration = std::chrono::milliseconds(static_cast<int>(base_seconds * 1000) + jitter_ms);

                std::unique_lock<std::mutex> lock(stop_mutex_);
                if (stop_cv_.wait_for(lock, sleep_duration, [this]() { return !running_.load(); }))
                {
                    break;
                }
            }
        }

        void HeartbeatReporter::SendHeartbeat()
        {
            // Lazy-init the CURL handle on first use so it stays alive across heartbeats.
            // curl_easy_reset() below reinitialises request-specific options while keeping
            // the connection pool, DNS cache, and TLS session cache alive — eliminating
            // the TCP/TLS handshake overhead on every heartbeat.
            if (!curl_handle_) {
                curl_handle_ = std::make_unique<detail::CurlHandle>();
            }
            curl_handle_->reset();

            if (!curl_handle_->is_valid()) {
                std::cerr << "[HeartbeatReporter] Failed to initialize CURL" << std::endl;
                consecutive_failures_++;
                return;
            }

            const auto& config = engine_->GetConfig();
            std::string url = config.platform_url + "/api/v1/edge-nodes/" + config.node_id + "/heartbeat";
            std::string payload = BuildHeartbeatPayload();

            curl_handle_->add_header("Content-Type: application/json");

            std::string auth_header = "Authorization: Bearer " + config.auth_token;
            curl_handle_->add_header(auth_header.c_str());

            std::string response_data;

            CURL* curl = curl_handle_->get();
            curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
            curl_easy_setopt(curl, CURLOPT_HTTPHEADER, curl_handle_->headers());
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, payload.c_str());
            curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, payload.length());

            curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, +[](void* contents, size_t size, size_t nmemb, void* userp) -> size_t {
                ((std::string*)userp)->append((char*)contents, size * nmemb);
                return size * nmemb;
            });
            curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response_data);
            curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 5L);
            curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

            CURLcode res = curl_easy_perform(curl);

            if (res == CURLE_OK)
            {
                long response_code = 0;
                curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &response_code);
                if (response_code >= 200 && response_code < 300)
                {
                    consecutive_failures_ = 0; // reset failure count on success
                    try {
                        ParseAndDeploy(response_data);
                    } catch (const std::exception& e) {
                        std::cerr << "[HeartbeatReporter] Failed to parse response: " << e.what() << std::endl;
                    }
                }
                else
                {
                    consecutive_failures_++;
                    std::cerr << "[HeartbeatReporter] Heartbeat failed with HTTP code " << response_code
                              << ": node_id=" << config.node_id << ", url=" << url << std::endl;
                }
            }
            else
            {
                consecutive_failures_++;
                std::cerr << "[HeartbeatReporter] Heartbeat failed: node_id=" << config.node_id
                          << ", url=" << url << ", error=" << curl_easy_strerror(res) << std::endl;
            }
        }

        std::string HeartbeatReporter::BuildHeartbeatPayload()
        {
            auto now = std::chrono::steady_clock::now();
            auto uptime = std::chrono::duration_cast<std::chrono::seconds>(now - start_time_).count();
            size_t load = engine_->GetPipelineManager()->ListPipelines().size();

            // Read enhanced metrics from DeviceMonitor via MetricsFlattener
            FlattenedMetrics flat_metrics;
            uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                std::chrono::system_clock::now().time_since_epoch()).count();
            uint64_t interval_ms = (last_flatten_timestamp_ms_ > 0)
                ? (now_ms - last_flatten_timestamp_ms_)
                : 5000;

            auto* device_monitor = engine_->GetDeviceMonitor();
            if (device_monitor)
            {
                auto snapshot_ptr = device_monitor->GetSnapshotPtr();
                if (snapshot_ptr)
                {
                    flat_metrics = metrics_flattener_->Flatten(*snapshot_ptr, interval_ms);
                }
            }

            // Read process counts directly
            auto proc_counts = CountProcesses();
            // Read network stats
            auto net_stats = ReadNetDev();
            // Read temperature
            double temperature = ReadTemperature();
            // Read uptime from /proc for accuracy
            uint64_t system_uptime = ReadUptime();

            last_flatten_timestamp_ms_ = now_ms;

            // Use flat_metrics values, falling back to direct reads
            double cpu_usage = flat_metrics.cpu_usage;
            double memory_usage = flat_metrics.memory_usage;
            double load_1m = flat_metrics.cpu_load_1m > 0 ? flat_metrics.cpu_load_1m : ReadLoad1m();

            // For network, use direct reads since they are most reliable
            uint64_t net_rx_bytes = net_stats.rx_bytes;
            uint64_t net_tx_bytes = net_stats.tx_bytes;

            double rx_speed = 0.0;
            double tx_speed = 0.0;
            if (last_net_timestamp_ms_ > 0 && now_ms > last_net_timestamp_ms_)
            {
                uint64_t elapsed_ms = now_ms - last_net_timestamp_ms_;
                if (net_rx_bytes >= last_net_rx_bytes_hb_)
                {
                    uint64_t rx_delta = net_rx_bytes - last_net_rx_bytes_hb_;
                    rx_speed = (static_cast<double>(rx_delta) * 1000.0) / static_cast<double>(elapsed_ms);
                }
                if (net_tx_bytes >= last_net_tx_bytes_hb_)
                {
                    uint64_t tx_delta = net_tx_bytes - last_net_tx_bytes_hb_;
                    tx_speed = (static_cast<double>(tx_delta) * 1000.0) / static_cast<double>(elapsed_ms);
                }
            }
            last_net_rx_bytes_hb_ = net_rx_bytes;
            last_net_tx_bytes_hb_ = net_tx_bytes;
            last_net_timestamp_ms_ = now_ms;

            json j;
            j["uptime"] = (system_uptime > 0 ? system_uptime : uptime);
            j["current_load"] = load;
            j["cpu_usage"] = cpu_usage;
            j["memory_usage"] = memory_usage;
            j["cpu_load_1m"] = load_1m;

            // Disk usage array
            json disks_arr = json::array();
            for (const auto& disk : flat_metrics.disk_usage)
            {
                json d;
                d["path"] = disk.path;
                d["total"] = disk.total;
                d["used"] = disk.used;
                d["percent"] = disk.percent;
                disks_arr.push_back(d);
            }
            j["disk_usage"] = disks_arr;

            // Network fields
            j["net_rx_bytes"] = net_rx_bytes;
            j["net_tx_bytes"] = net_tx_bytes;
            j["net_rx_speed"] = rx_speed;
            j["net_tx_speed"] = tx_speed;
            j["temperature"] = temperature;
            j["process_count"] = proc_counts.processes;
            j["thread_count"] = proc_counts.threads;
            j["engine_version"] = InferenceEngine::Version();
            j["hal_platform"] = engine_->GetHalPlatform();

            // Basic hardware info
            json hw;
            hw["cpu_model"] = GetCPUModel();
            hw["total_memory"] = GetTotalMemory();
            hw["cpu_cores"] = 4;
            j["hardware_info"] = hw;

            // Installed algorithms
            json algs_arr = json::array();
            auto deployments = engine_->GetAlgoManager()->GetDeployments();
            for (const auto& dep : deployments)
            {
                std::string runtime_status =
                    engine_->GetAlgoManager()->IsLoaded(dep.algo_name) ? "ready" : "installed";
                json alg;
                alg["algo_package_id"] = dep.algo_package_id;
                alg["algo_name"] = dep.algo_name;
                alg["version"] = dep.version;
                alg["install_path"] = dep.install_path;
                alg["status"] = dep.status;
                alg["runtime_status"] = runtime_status;
                alg["supports_embedding"] = true;
                alg["supports_face_library"] = true;
                alg["embedding_capacity"] = 1;
                algs_arr.push_back(alg);
            }
            j["installed_algorithms"] = algs_arr;

            // Engine-specific metrics
            auto* metrics_reporter = engine_->GetMetricsReporter();
            if (metrics_reporter)
            {
                auto engine_metrics = metrics_reporter->CollectNow();
                j["worker_count"] = engine_metrics.worker_count;
                j["idle_worker_count"] = engine_metrics.idle_worker_count;
                j["active_stream_count"] = engine_metrics.active_stream_count;
                j["decode_sessions"] = engine_metrics.decode_sessions;
                j["encode_sessions"] = engine_metrics.encode_sessions;
            }

            return j.dump();
        }

        void HeartbeatReporter::ParseAndDeploy(const std::string& response_json)
        {
            try {
                // Parse JSON response using nlohmann/json library
                auto response = json::parse(response_json);

                // The Go backend returns "code" as a string — "OK" for success,
                // or an error code like "NODE_NOT_FOUND", "ENGINE_VERSION_INCOMPATIBLE" etc.
                if (!response.contains("code") || !response["code"].is_string()) {
                    std::cerr << "[HeartbeatReporter] Heartbeat response missing or invalid 'code' field" << std::endl;
                    return;
                }
                const std::string code = response["code"].get<std::string>();
                if (code != "OK") {
                    std::cerr << "[HeartbeatReporter] Heartbeat response error: code=" << code
                              << ", message=" << (response.contains("message") ? response["message"].get<std::string>() : "unknown")
                              << std::endl;
                    return;
                }

                // Extract pending deployments array
                if (!response.contains("data") || !response["data"].contains("pending_deployments")) {
                    return; // No pending deployments
                }

                const auto& pending_deployments = response["data"]["pending_deployments"];
                if (!pending_deployments.is_array()) {
                    std::cerr << "[HeartbeatReporter] Invalid pending_deployments format" << std::endl;
                    return;
                }

                // Process each deployment
                for (const auto& deployment : pending_deployments) {
                    // Validate required fields
                    if (!deployment.contains("algo_package_id") || !deployment.contains("download_url") ||
                        !deployment.contains("md5") || !deployment.contains("extract_path") ||
                        !deployment.contains("algo_name") || !deployment.contains("version")) {
                        std::cerr << "[HeartbeatReporter] Skipping deployment with missing fields" << std::endl;
                        continue;
                    }

                    std::string pkg_id = deployment["algo_package_id"].get<std::string>();
                    std::string url = deployment["download_url"].get<std::string>();
                    std::string md5 = deployment["md5"].get<std::string>();
                    std::string path = deployment["extract_path"].get<std::string>();
                    std::string name = deployment["algo_name"].get<std::string>();
                    std::string version = deployment["version"].get<std::string>();

                    // Validate non-empty values
                    if (pkg_id.empty() || url.empty() || md5.empty() ||
                        path.empty() || name.empty() || version.empty()) {
                        std::cerr << "[HeartbeatReporter] Skipping deployment with empty fields" << std::endl;
                        continue;
                    }

                    std::cout << "[HeartbeatReporter] Scheduling deploy for: " << name
                              << " (package " << pkg_id << ")" << std::endl;

                    algo::AlgorithmDownloader::StartDeploy(
                        engine_->GetAlgoManager(),
                        url,
                        md5,
                        path,
                        pkg_id,
                        name,
                        version,
                        engine_->GetConfig().algo_dir
                    );
                }
            } catch (const json::parse_error& e) {
                std::cerr << "[HeartbeatReporter] JSON parse error: " << e.what()
                          << " at byte " << e.byte << std::endl;
            } catch (const json::type_error& e) {
                std::cerr << "[HeartbeatReporter] JSON type error: " << e.what() << std::endl;
            } catch (const std::exception& e) {
                std::cerr << "[HeartbeatReporter] Unexpected error parsing response: " << e.what() << std::endl;
            }
        }

    } // namespace monitor
} // namespace aivision
