#include "probes/device_probe.h"

#include <fstream>
#include <sstream>
#include <algorithm>
#include <iostream>
#include <chrono>
#include <filesystem>

namespace aivision
{
    namespace monitor
    {
        class RockchipProbe : public IDeviceProbe
        {
        public:
            RockchipProbe() = default;
            ~RockchipProbe() override = default;

            std::string Name() const override { return "rockchip"; }

            ProbeDetection Detect(const DeviceMonitorConfig& config) override
            {
                ProbeDetection det;
#ifdef __APPLE__
                det.supported = false;
                det.confidence = 0;
                det.evidence.push_back("Platform is macOS (Darwin)");
#else
                bool found_compatible = false;
                std::ifstream comp_file("/proc/device-tree/compatible");
                if (comp_file.is_open())
                {
                    std::string line;
                    if (std::getline(comp_file, line))
                    {
                        std::transform(line.begin(), line.end(), line.begin(), ::tolower);
                        if (line.find("rockchip") != std::string::npos)
                        {
                            found_compatible = true;
                            det.evidence.push_back("compatible: " + line);
                        }
                    }
                }

                bool found_model = false;
                std::ifstream model_file("/proc/device-tree/model");
                if (model_file.is_open())
                {
                    std::string line;
                    if (std::getline(model_file, line))
                    {
                        std::transform(line.begin(), line.end(), line.begin(), ::tolower);
                        if (line.find("rockchip") != std::string::npos || 
                            line.find("rk3588") != std::string::npos ||
                            line.find("rk3568") != std::string::npos)
                        {
                            found_model = true;
                            det.evidence.push_back("model: " + line);
                        }
                    }
                }

                if (found_compatible || found_model)
                {
                    det.supported = true;
                    det.confidence = 90;
                    det.evidence.push_back("Rockchip signature found in device tree");
                }
                else
                {
                    det.supported = false;
                    det.confidence = 0;
                    det.evidence.push_back("No Rockchip signature found in device tree");
                }
#endif
                return det;
            }

            void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) override
            {
#ifndef __APPLE__
                std::string model;
                std::ifstream model_file("/proc/device-tree/model");
                if (model_file.is_open())
                {
                    std::getline(model_file, model);
                    // Remove trailing null-byte if any
                    if (!model.empty() && model.back() == '\0') {
                        model.pop_back();
                    }
                }

                std::string comp;
                std::ifstream comp_file("/proc/device-tree/compatible");
                if (comp_file.is_open())
                {
                    std::getline(comp_file, comp);
                }

                std::transform(comp.begin(), comp.end(), comp.begin(), ::tolower);
                std::transform(model.begin(), model.end(), model.begin(), ::tolower);

                if (!model.empty())
                {
                    info.board = model;
                }

                if (comp.find("rk3588") != std::string::npos || model.find("rk3588") != std::string::npos)
                {
                    info.gpu_model = "Mali-G610";
                    info.npu_model = "RKNPU v2";
                }
                else if (comp.find("rk3568") != std::string::npos || model.find("rk3568") != std::string::npos)
                {
                    info.gpu_model = "Mali-G52";
                    info.npu_model = "RKNPU v1";
                }
                else
                {
                    info.gpu_model = "Mali GPU";
                    info.npu_model = "RKNPU";
                }
#endif
            }

            void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifndef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                // 1. Temperature Detection
                // Scans thermal zones prioritising soc/cpu/gpu/npu
                std::string best_zone_path;
                int best_priority = -1;

                for (int i = 0; i < 20; ++i)
                {
                    std::string zone_dir = "/sys/class/thermal/thermal_zone" + std::to_string(i);
                    if (!std::filesystem::exists(zone_dir))
                        break;

                    std::ifstream type_file(zone_dir + "/type");
                    if (type_file.is_open())
                    {
                        std::string type;
                        std::getline(type_file, type);
                        std::transform(type.begin(), type.end(), type.begin(), ::tolower);

                        int priority = -1;
                        if (type.find("soc") != std::string::npos) priority = 4;
                        else if (type.find("cpu") != std::string::npos) priority = 3;
                        else if (type.find("gpu") != std::string::npos) priority = 2;
                        else if (type.find("npu") != std::string::npos) priority = 1;

                        if (priority > best_priority)
                        {
                            best_priority = priority;
                            best_zone_path = zone_dir + "/temp";
                        }
                    }
                }

                if (!best_zone_path.empty())
                {
                    std::ifstream temp_file(best_zone_path);
                    if (temp_file.is_open())
                    {
                        double raw_temp = 0;
                        if (temp_file >> raw_temp)
                        {
                            metrics.temperature.available = true;
                            metrics.temperature.stale = false;
                            metrics.temperature.value = raw_temp / 1000.0;
                            metrics.temperature.unit = "celsius";
                            metrics.temperature.source = best_zone_path;
                            metrics.temperature.error = "";
                            metrics.temperature.collected_at_ms = now_ms;
                        }
                    }
                }

                // 2. NPU usage from debugfs /sys/kernel/debug/rknpu/load
                std::ifstream npu_load_file("/sys/kernel/debug/rknpu/load");
                if (npu_load_file.is_open())
                {
                    std::string line;
                    AcceleratorInfo npu;
                    npu.type = "npu";
                    npu.vendor = "rockchip";
                    npu.name = "RKNPU";
                    
                    std::vector<double> core_loads;

                    if (std::getline(npu_load_file, line))
                    {
                        // Format: "NPU load: Core0: 25%, Core1: 30%, Core2: 10%"
                        size_t pos = 0;
                        while ((pos = line.find("Core", pos)) != std::string::npos)
                        {
                            size_t colon = line.find(":", pos);
                            if (colon != std::string::npos)
                            {
                                size_t percent = line.find("%", colon);
                                if (percent != std::string::npos)
                                {
                                    std::string core_name = line.substr(pos, colon - pos);
                                    std::string val_str = line.substr(colon + 1, percent - colon - 1);
                                    try
                                    {
                                        double val = std::stod(val_str);
                                        AcceleratorCoreMetric core;
                                        core.name = core_name;
                                        core.usage = val;
                                        npu.cores.push_back(core);
                                        core_loads.push_back(val);
                                    }
                                    catch (...) {}
                                }
                            }
                            pos += 4;
                        }

                        // Fallback single-core parsing
                        if (npu.cores.empty())
                        {
                            size_t percent = line.find("%");
                            if (percent != std::string::npos)
                            {
                                size_t start = line.find_last_of(" :", percent);
                                if (start == std::string::npos) start = 0;
                                else start++;
                                std::string val_str = line.substr(start, percent - start);
                                try
                                {
                                    double val = std::stod(val_str);
                                    AcceleratorCoreMetric core;
                                    core.name = "Core0";
                                    core.usage = val;
                                    npu.cores.push_back(core);
                                    core_loads.push_back(val);
                                }
                                catch (...) {}
                            }
                        }
                    }

                    if (!core_loads.empty())
                    {
                        double sum = 0.0;
                        for (double val : core_loads)
                        {
                            sum += val;
                        }
                        double clamped_sum = std::min(100.0, sum);

                        npu.usage.available = true;
                        npu.usage.stale = false;
                        npu.usage.value = clamped_sum;
                        npu.usage.unit = "percent";
                        npu.usage.source = "/sys/kernel/debug/rknpu/load";
                        npu.usage.error = "";
                        npu.usage.collected_at_ms = now_ms;

                        // Temp is optional but if we have zone we can fill it
                        if (metrics.temperature.available)
                        {
                            npu.temperature = metrics.temperature;
                        }

                        metrics.npu_usage = npu.usage;

                        // Update in accelerators array
                        auto it = std::find_if(accelerators.begin(), accelerators.end(), [](const AcceleratorInfo& acc) {
                            return acc.vendor == "rockchip" && acc.type == "npu";
                        });
                        if (it != accelerators.end())
                        {
                            *it = npu;
                        }
                        else
                        {
                            accelerators.push_back(npu);
                        }
                    }
                    else
                    {
                        metrics.npu_usage.available = false;
                        metrics.npu_usage.error = "Parse rknpu load output failed";
                    }
                }
                else
                {
                    metrics.npu_usage.available = false;
                    metrics.npu_usage.error = "Failed to open rknpu debugfs load file (debugfs not mounted?)";
                }
#endif
            }

            void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
                // Rockchip doesn't have expensive CLI collection in debugfs mode
            }
        };

        std::shared_ptr<IDeviceProbe> CreateRockchipProbe()
        {
            return std::make_shared<RockchipProbe>();
        }
    }
}
