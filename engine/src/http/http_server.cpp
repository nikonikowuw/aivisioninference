#include "http/http_server.h"
#include "engine.h"
#include "algo/algo_manager.h"
#include "algo/algorithm_downloader.h"
#include "pipeline/pipeline_manager.h"
#include <httplib.h>
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

namespace aivision
{
    namespace http
    {
        namespace
        {
            std::string ExtractJsonField(const std::string &json, const std::string &field_name)
            {
                std::string search = "\"" + field_name + "\"";
                size_t pos = json.find(search);
                if (pos == std::string::npos)
                    return "";

                size_t start = json.find(":", pos);
                if (start == std::string::npos)
                    return "";

                start++;
                while (start < json.size() && (json[start] == ' ' || json[start] == '\t' || json[start] == '\r' || json[start] == '\n'))
                    start++;

                if (start >= json.size())
                    return "";

                if (json[start] == '"')
                {
                    start++;
                    std::string value;
                    bool escaped = false;
                    for (size_t i = start; i < json.size(); ++i)
                    {
                        char ch = json[i];
                        if (escaped)
                        {
                            switch (ch)
                            {
                            case 'n': value.push_back('\n'); break;
                            case 'r': value.push_back('\r'); break;
                            case 't': value.push_back('\t'); break;
                            default: value.push_back(ch); break;
                            }
                            escaped = false;
                            continue;
                        }
                        if (ch == '\\')
                        {
                            escaped = true;
                            continue;
                        }
                        if (ch == '"')
                        {
                            return value;
                        }
                        value.push_back(ch);
                    }
                    return "";
                }

                size_t end = json.find_first_of(",}\n", start);
                if (end == std::string::npos)
                    end = json.size();
                
                std::string val = json.substr(start, end - start);
                // Trim trailing whitespace/brackets
                while (!val.empty() && (val.back() == ' ' || val.back() == '\t' || val.back() == '\r' || val.back() == '\n' || val.back() == '}'))
                {
                    val.pop_back();
                }
                return val;
            }

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
                return 16ULL * 1024 * 1024 * 1024; // 16GB default
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
                return 8ULL * 1024 * 1024 * 1024; // 8GB default
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

        HTTPServer::HTTPServer(InferenceEngine* engine)
            : engine_(engine)
        {
        }

        HTTPServer::~HTTPServer()
        {
            Stop();
        }

        bool HTTPServer::Start(int port)
        {
            if (running_.load())
                return false;

            port_ = port;
            running_.store(true);
            thread_ = std::make_unique<std::thread>(&HTTPServer::RunServer, this, port);
            return true;
        }

        void HTTPServer::Stop()
        {
            if (!running_.load())
                return;

            running_.store(false);
            
            void* ptr = svr_ptr_.load();
            if (ptr)
            {
                static_cast<httplib::Server*>(ptr)->stop();
            }
            
            // To force stop httplib server, we can make a dummy request or rely on thread join
            // since we run svr.listen in a loop checking running_ status or svr.stop()
            if (thread_ && thread_->joinable())
            {
                thread_->join();
            }
            thread_.reset();
            svr_ptr_.store(nullptr);
        }

        void HTTPServer::RunServer(int port)
        {
            httplib::Server svr;
            svr_ptr_.store(&svr);
            
            auto startup_time = std::chrono::steady_clock::now();

            // Set CORS Headers post-routing
            svr.set_post_routing_handler([](const httplib::Request&, httplib::Response& res) {
                res.set_header("Access-Control-Allow-Origin", "*");
                res.set_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
                res.set_header("Access-Control-Allow-Headers", "Content-Type, Authorization");
            });

            // OPTIONS preflight
            svr.Options(R"(.*)", [](const httplib::Request&, httplib::Response& res) {
                res.set_header("Access-Control-Allow-Origin", "*");
                res.set_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
                res.set_header("Access-Control-Allow-Headers", "Content-Type, Authorization");
                res.status = 200;
            });

            // 1. Health check
            svr.Get("/health", [this, startup_time](const httplib::Request&, httplib::Response& res) {
                auto now = std::chrono::steady_clock::now();
                auto uptime = std::chrono::duration_cast<std::chrono::seconds>(now - startup_time).count();
                size_t load = engine_->GetPipelineManager()->ListPipelines().size();

                std::stringstream ss;
                ss << "{"
                   << "\"status\":\"ok\","
                   << "\"uptime\":" << uptime << ","
                   << "\"current_load\":" << load << ","
                   << "\"engine_version\":\"" << InferenceEngine::Version() << "\""
                   << "}";

                res.set_content(ss.str(), "application/json");
            });

            // 2. Hardware info
            svr.Get("/hardware-info", [](const httplib::Request&, httplib::Response& res) {
#ifdef __APPLE__
                std::string platform = "macos";
                std::string gpu = "Apple M1 GPU";
#else
                std::string platform = "rkmpp";
                std::string gpu = "Mali-G52";
#endif
                std::stringstream ss;
                ss << "{"
                   << "\"hal_platform\":\"" << platform << "\","
                   << "\"cpu_model\":\"" << GetCPUModel() << "\","
                   << "\"gpu_model\":\"" << gpu << "\","
                   << "\"total_memory\":" << GetTotalMemory()
                   << "}";

                res.set_content(ss.str(), "application/json");
            });

            // 3. Deploy algorithm
            svr.Post("/deploy-algo", [this](const httplib::Request& req, httplib::Response& res) {
                std::string body = req.body;
                std::string pkg_id = ExtractJsonField(body, "algo_package_id");
                std::string url = ExtractJsonField(body, "download_url");
                std::string md5 = ExtractJsonField(body, "md5");
                std::string path = ExtractJsonField(body, "extract_path");
                std::string name = ExtractJsonField(body, "algo_name");
                std::string version = ExtractJsonField(body, "version");

                if (pkg_id.empty() || url.empty() || md5.empty() || path.empty() || name.empty() || version.empty())
                {
                    res.status = 400;
                    res.set_content("{\"success\":false,\"error_message\":\"Missing required fields\"}", "application/json");
                    return;
                }

                std::cout << "[HTTPServer] Received deploy-algo for package: " << pkg_id << std::endl;

                // Trigger async deployment
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

                res.status = 200;
                res.set_content("{\"success\":true,\"message\":\"Deployment triggered\"}", "application/json");
            });

            // 4. List algorithms
            svr.Get("/algorithms", [this](const httplib::Request&, httplib::Response& res) {
                auto instances = engine_->GetAlgoManager()->GetAllInstances();
                auto deployments = engine_->GetAlgoManager()->GetDeployments();

                std::stringstream ss;
                ss << "[";
                bool first = true;

                // Add loaded algorithms
                for (const auto& inst : instances)
                {
                    if (!first) ss << ",";
                    first = false;
                    ss << "{"
                       << "\"algo_name\":\"" << inst->GetName() << "\","
                       << "\"version\":\"" << inst->GetVersion() << "\","
                       << "\"status\":\"installed\""
                       << "}";
                }

                // Add active or failed deployments
                for (const auto& dep : deployments)
                {
                    // If already loaded, skip it to avoid duplication
                    bool loaded = false;
                    for (const auto& inst : instances)
                    {
                        if (inst->GetName() == dep.algo_package_id || inst->GetVersion() == dep.version)
                        {
                            loaded = true;
                            break;
                        }
                    }
                    if (loaded && dep.status == "installed")
                        continue;

                    if (!first) ss << ",";
                    first = false;
                    ss << "{"
                       << "\"algo_name\":\"" << dep.algo_package_id << "\","
                       << "\"version\":\"" << dep.version << "\","
                       << "\"status\":\"" << dep.status << "\","
                       << "\"install_path\":\"" << dep.install_path << "\","
                       << "\"error_message\":\"" << dep.error_message << "\""
                       << "}";
                }

                ss << "]";
                res.set_content(ss.str(), "application/json");
            });

            std::cout << "[HTTPServer] Listening on port " << port << std::endl;
            svr.listen("0.0.0.0", port);
            
            svr_ptr_.store(nullptr);
        }

    } // namespace http
} // namespace aivision
