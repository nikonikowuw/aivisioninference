#include "monitor/heartbeat_reporter.h"
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

#ifdef __APPLE__
#include <sys/types.h>
#include <sys/sysctl.h>
#else
#include <sys/sysinfo.h>
#endif

using json = nlohmann::json;

namespace aivision
{
    namespace monitor
    {
        namespace
        {
            // RAII wrapper for CURL resources to ensure cleanup on exception/early return
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
            };

            uint64_t GetTotalMemory()
            {
#ifdef __APPLE__
                int mib[2] = {CTL_HW, HW_MEMSIZE};
                int64_t physical_memory = 0;
                size_t length = sizeof(physical_memory);
                if (sysctl(mib, 2, &physical_memory, &length, NULL, 0) == 0)
                {
                    return physical_memory;
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
            }

            std::string GetCPUModel()
            {
#ifdef __APPLE__
                char buffer[256];
                size_t buffer_len = sizeof(buffer);
                if (sysctlbyname("machdep.cpu.brand_string", &buffer, &buffer_len, NULL, 0) == 0)
                {
                    return std::string(buffer);
                }
                return "Apple Silicon";
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
                                return line.substr(colon + 1);
                            }
                        }
                    }
                }
                return "ARM Cortex-A55";
#endif
            }
        }

        HeartbeatReporter::HeartbeatReporter(InferenceEngine* engine)
            : engine_(engine)
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
            while (running_.load())
            {
                SendHeartbeat();

                std::unique_lock<std::mutex> lock(stop_mutex_);
                if (stop_cv_.wait_for(lock, std::chrono::seconds(5), [this]() { return !running_.load(); }))
                {
                    break;
                }
            }
        }

        void HeartbeatReporter::SendHeartbeat()
        {
            CurlHandle curl_handle;
            if (!curl_handle.is_valid()) {
                std::cerr << "[HeartbeatReporter] Failed to initialize CURL" << std::endl;
                return;
            }

            const auto& config = engine_->GetConfig();
            std::string url = config.platform_url + "/api/v1/edge-nodes/" + config.node_id + "/heartbeat";
            std::string payload = BuildHeartbeatPayload();

            curl_handle.add_header("Content-Type: application/json");
            
            std::string auth_header = "Authorization: Bearer " + config.auth_token;
            curl_handle.add_header(auth_header.c_str());

            std::string response_data;

            CURL* curl = curl_handle.get();
            curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
            curl_easy_setopt(curl, CURLOPT_HTTPHEADER, curl_handle.headers());
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, payload.c_str());
            curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, payload.length());
            
            curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, +[](void* contents, size_t size, size_t nmemb, void* userp) -> size_t {
                ((std::string*)userp)->append((char*)contents, size * nmemb);
                return size * nmemb;
            });
            curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response_data);
            curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

            CURLcode res = curl_easy_perform(curl);
            
            // Resources are automatically cleaned up by CurlHandle destructor
            
            if (res == CURLE_OK)
            {
                try {
                    ParseAndDeploy(response_data);
                } catch (const std::exception& e) {
                    std::cerr << "[HeartbeatReporter] Failed to parse response: " << e.what() << std::endl;
                }
            }
            else
            {
                std::cerr << "[HeartbeatReporter] Heartbeat failed: node_id=" << config.node_id
                          << ", url=" << url << ", error=" << curl_easy_strerror(res) << std::endl;
            }
        }

        std::string HeartbeatReporter::BuildHeartbeatPayload()
        {
            auto now = std::chrono::steady_clock::now();
            auto uptime = std::chrono::duration_cast<std::chrono::seconds>(now - start_time_).count();
            size_t load = engine_->GetPipelineManager()->ListPipelines().size();

            // Build installed algorithms JSON array string
            std::string algo_json_str = "[]";
            try {
                json algo_arr = json::array();
                auto deployments = engine_->GetAlgoManager()->GetDeployments();
                for (const auto& dep : deployments)
                {
                    const std::string runtime_status =
                        engine_->GetAlgoManager()->IsLoaded(dep.algo_name) ? "ready" : "installed";
                    json a;
                    a["algo_package_id"] = dep.algo_package_id;
                    a["algo_name"] = dep.algo_name;
                    a["version"] = dep.version;
                    a["install_path"] = dep.install_path;
                    a["status"] = dep.status;
                    a["runtime_status"] = runtime_status;
                    a["supports_embedding"] = true;
                    a["supports_face_library"] = true;
                    a["embedding_capacity"] = 1;
                    algo_arr.push_back(a);
                }
                algo_json_str = algo_arr.dump();
            } catch (...) {
                algo_json_str = "[]";
            }

            // If DeviceMonitor is available, use ToHeartbeatJson with real snapshot data
            if (device_monitor_) {
                auto snapshot_ptr = device_monitor_->GetSnapshotPtr();
                if (snapshot_ptr) {
                    return ToHeartbeatJson(uptime, load, InferenceEngine::Version(),
                        engine_->GetHalPlatform(), algo_json_str, *snapshot_ptr);
                }
            }

            // Fallback: build minimal heartbeat without DeviceMonitor
            std::stringstream ss;
            ss << "{"
               << "\"uptime\":" << uptime << ","
               << "\"current_load\":" << load << ","
               << "\"cpu_usage\":" << 0.0 << ","
               << "\"memory_usage\":" << 0.0 << ","
               << "\"engine_version\":\"" << InferenceEngine::Version() << "\","
               << "\"hal_platform\":\"" << engine_->GetHalPlatform() << "\"";

            // Basic hardware info
            ss << ",\"hardware_info\":{"
               << "\"cpu_model\":\"" << GetCPUModel() << "\","
               << "\"total_memory\":" << GetTotalMemory() << ","
               << "\"cpu_cores\":4"
               << "}";

            // Installed algorithms
            ss << ",\"installed_algorithms\":" << algo_json_str;
            ss << "}";
            return ss.str();
        }

        void HeartbeatReporter::ParseAndDeploy(const std::string& response_json)
        {
            try {
                // Parse JSON response using nlohmann/json library
                auto response = json::parse(response_json);
                
                // Check for successful response structure
                if (!response.contains("code") || response["code"].get<int>() != 0) {
                    std::cerr << "[HeartbeatReporter] Heartbeat response error: "
                              << (response.contains("message") ? response["message"].get<std::string>() : "unknown")
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
