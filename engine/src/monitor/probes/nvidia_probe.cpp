#include "probes/device_probe.h"

#include <fstream>
#include <sstream>
#include <algorithm>
#include <iostream>
#include <chrono>
#include <filesystem>
#include <unistd.h>

namespace aivision
{
    namespace monitor
    {
        namespace
        {
            bool HasNvidiaSmi()
            {
                if (access("/usr/bin/nvidia-smi", X_OK) == 0) return true;
                if (access("/usr/local/bin/nvidia-smi", X_OK) == 0) return true;
                auto res = CommandRunner::Run({"which", "nvidia-smi"}, 1000);
                return res.success;
            }

            bool IsJetson()
            {
                return std::filesystem::exists("/etc/nv_tegra_release") || 
                       std::filesystem::exists("/sys/devices/gpu.0/load");
            }

            std::string TrimString(const std::string& str)
            {
                size_t first = str.find_first_not_of(" \t\r\n");
                if (first == std::string::npos) return "";
                size_t last = str.find_last_not_of(" \t\r\n");
                return str.substr(first, last - first + 1);
            }
        } // namespace

        class NvidiaProbe : public IDeviceProbe
        {
        public:
            NvidiaProbe() = default;
            ~NvidiaProbe() override = default;

            std::string Name() const override { return "nvidia"; }

            ProbeDetection Detect(const DeviceMonitorConfig& config) override
            {
                ProbeDetection det;
#ifdef __APPLE__
                det.supported = false;
                det.confidence = 0;
                det.evidence.push_back("Platform is macOS (Darwin)");
#else
                if (HasNvidiaSmi())
                {
                    det.supported = true;
                    det.confidence = 90;
                    det.evidence.push_back("nvidia-smi found in system path");
                }
                else if (IsJetson())
                {
                    det.supported = true;
                    det.confidence = 95;
                    det.evidence.push_back("Jetson platform signatures found");
                }
                else
                {
                    det.supported = false;
                    det.confidence = 0;
                    det.evidence.push_back("No NVIDIA GPU evidence detected");
                }
#endif
                return det;
            }

            void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) override
            {
#ifndef __APPLE__
                if (IsJetson())
                {
                    info.gpu_model = "NVIDIA Tegra GPU";
                    info.board = "NVIDIA Jetson Board";
                }
                else if (HasNvidiaSmi() && config.enable_external_commands)
                {
                    // Query model name via nvidia-smi
                    auto res = CommandRunner::Run({"nvidia-smi", "--query-gpu=name", "--format=csv,noheader"}, config.command_timeout_ms);
                    if (res.success && !res.stdout_data.empty())
                    {
                        info.gpu_model = TrimString(res.stdout_data);
                    }
                }
#endif
            }

            void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifndef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                if (IsJetson())
                {
                    // Read Jetson load in ppt (parts-per-thousand)
                    std::ifstream load_file("/sys/devices/gpu.0/load");
                    if (load_file.is_open())
                    {
                        double ppt = 0;
                        if (load_file >> ppt)
                        {
                            metrics.gpu_usage.available = true;
                            metrics.gpu_usage.stale = false;
                            metrics.gpu_usage.value = ppt / 10.0;
                            metrics.gpu_usage.unit = "percent";
                            metrics.gpu_usage.source = "/sys/devices/gpu.0/load";
                            metrics.gpu_usage.error = "";
                            metrics.gpu_usage.collected_at_ms = now_ms;

                            // Update accelerator array
                            AcceleratorInfo gpu;
                            gpu.type = "gpu";
                            gpu.vendor = "nvidia";
                            gpu.name = "NVIDIA Jetson GPU";
                            gpu.usage = metrics.gpu_usage;
                            gpu.memory_usage.available = false;
                            gpu.memory_usage.error = "Unified memory";

                            // Try to get temperature from thermal zone if possible
                            for (int i = 0; i < 10; ++i)
                            {
                                std::string type_path = "/sys/class/thermal/thermal_zone" + std::to_string(i) + "/type";
                                if (std::filesystem::exists(type_path))
                                {
                                    std::ifstream type_file(type_path);
                                    std::string type;
                                    std::getline(type_file, type);
                                    std::transform(type.begin(), type.end(), type.begin(), ::tolower);
                                    if (type.find("gpu") != std::string::npos)
                                    {
                                        std::ifstream temp_file("/sys/class/thermal/thermal_zone" + std::to_string(i) + "/temp");
                                        double temp = 0;
                                        if (temp_file >> temp)
                                        {
                                            gpu.temperature.available = true;
                                            gpu.temperature.stale = false;
                                            gpu.temperature.value = temp / 1000.0;
                                            gpu.temperature.unit = "celsius";
                                            gpu.temperature.source = type_path;
                                            gpu.temperature.error = "";
                                            gpu.temperature.collected_at_ms = now_ms;
                                            break;
                                        }
                                    }
                                }
                            }

                            auto it = std::find_if(accelerators.begin(), accelerators.end(), [](const AcceleratorInfo& acc) {
                                return acc.vendor == "nvidia" && acc.type == "gpu";
                            });
                            if (it != accelerators.end())
                            {
                                *it = gpu;
                            }
                            else
                            {
                                accelerators.push_back(gpu);
                            }
                        }
                    }
                }
#endif
            }

            void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifndef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                if (HasNvidiaSmi() && config.enable_external_commands)
                {
                    // Run nvidia-smi query
                    std::vector<std::string> args = {
                        "nvidia-smi",
                        "--query-gpu=name,utilization.gpu,memory.total,memory.used,temperature.gpu",
                        "--format=csv,noheader,nounits"
                    };

                    auto res = CommandRunner::Run(args, config.command_timeout_ms);
                    if (res.success && !res.stdout_data.empty())
                    {
                        std::vector<AcceleratorInfo> nvidia_gpus;
                        double max_usage = 0.0;
                        bool has_gpus = false;

                        std::stringstream ss(res.stdout_data);
                        std::string line;
                        while (std::getline(ss, line))
                        {
                            if (line.empty()) continue;

                            std::stringstream line_ss(line);
                            std::string name_part, util_part, mem_total_part, mem_used_part, temp_part;

                            std::getline(line_ss, name_part, ',');
                            std::getline(line_ss, util_part, ',');
                            std::getline(line_ss, mem_total_part, ',');
                            std::getline(line_ss, mem_used_part, ',');
                            std::getline(line_ss, temp_part, ',');

                            name_part = TrimString(name_part);
                            util_part = TrimString(util_part);
                            mem_total_part = TrimString(mem_total_part);
                            mem_used_part = TrimString(mem_used_part);
                            temp_part = TrimString(temp_part);

                            try
                            {
                                double util = std::stod(util_part);
                                double total_mem = std::stod(mem_total_part);
                                double used_mem = std::stod(mem_used_part);
                                double temp = std::stod(temp_part);

                                AcceleratorInfo gpu;
                                gpu.type = "gpu";
                                gpu.vendor = "nvidia";
                                gpu.name = name_part;

                                gpu.usage.available = true;
                                gpu.usage.stale = false;
                                gpu.usage.value = util;
                                gpu.usage.unit = "percent";
                                gpu.usage.source = "nvidia-smi";
                                gpu.usage.collected_at_ms = now_ms;

                                if (total_mem > 0)
                                {
                                    gpu.memory_usage.available = true;
                                    gpu.memory_usage.stale = false;
                                    gpu.memory_usage.value = (used_mem / total_mem) * 100.0;
                                    gpu.memory_usage.unit = "percent";
                                    gpu.memory_usage.source = "nvidia-smi";
                                    gpu.memory_usage.collected_at_ms = now_ms;
                                }
                                else
                                {
                                    gpu.memory_usage.available = false;
                                    gpu.memory_usage.error = "Zero GPU memory total";
                                }

                                gpu.temperature.available = true;
                                gpu.temperature.stale = false;
                                gpu.temperature.value = temp;
                                gpu.temperature.unit = "celsius";
                                gpu.temperature.source = "nvidia-smi";
                                gpu.temperature.collected_at_ms = now_ms;

                                nvidia_gpus.push_back(gpu);
                                max_usage = std::max(max_usage, util);
                                has_gpus = true;
                            }
                            catch (...) {}
                        }

                        if (has_gpus)
                        {
                            metrics.gpu_usage.available = true;
                            metrics.gpu_usage.stale = false;
                            metrics.gpu_usage.value = max_usage;
                            metrics.gpu_usage.unit = "percent";
                            metrics.gpu_usage.source = "nvidia-smi (max)";
                            metrics.gpu_usage.collected_at_ms = now_ms;

                            // Update the accelerators list, removing any old nvidia GPUs
                            accelerators.erase(
                                std::remove_if(accelerators.begin(), accelerators.end(), 
                                               [](const AcceleratorInfo& acc) {
                                                   return acc.vendor == "nvidia";
                                               }),
                                accelerators.end()
                            );

                            for (const auto& gpu : nvidia_gpus)
                            {
                                accelerators.push_back(gpu);
                            }
                        }
                    }
                    else
                    {
                        metrics.gpu_usage.available = false;
                        metrics.gpu_usage.error = "nvidia-smi execution failed: " + res.error;
                        throw std::runtime_error("nvidia-smi collection failed: " + res.error);
                    }
                }
#endif
            }
        };

        std::shared_ptr<IDeviceProbe> CreateNvidiaProbe()
        {
            return std::make_shared<NvidiaProbe>();
        }
    }
}
