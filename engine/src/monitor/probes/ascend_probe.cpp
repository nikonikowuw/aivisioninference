#include "probes/device_probe.h"

#include <fstream>
#include <sstream>
#include <algorithm>
#include <iostream>
#include <chrono>
#include <filesystem>
#include <unistd.h>
#include <map>

namespace aivision
{
    namespace monitor
    {
        namespace
        {
            bool HasNpuSmi()
            {
                if (access("/usr/local/bin/npu-smi", X_OK) == 0) return true;
                if (access("/usr/bin/npu-smi", X_OK) == 0) return true;
                auto res = CommandRunner::Run({"which", "npu-smi"}, 1000);
                return res.success;
            }

            std::string Trim(const std::string& str)
            {
                size_t first = str.find_first_not_of(" \t\r\n");
                if (first == std::string::npos) return "";
                size_t last = str.find_last_not_of(" \t\r\n");
                return str.substr(first, last - first + 1);
            }

            std::vector<std::string> Split(const std::string& str, char delim)
            {
                std::vector<std::string> tokens;
                std::stringstream ss(str);
                std::string token;
                while (std::getline(ss, token, delim))
                {
                    tokens.push_back(token);
                }
                return tokens;
            }
        } // namespace

        class AscendProbe : public IDeviceProbe
        {
        public:
            AscendProbe() = default;
            ~AscendProbe() override = default;

            std::string Name() const override { return "ascend"; }

            ProbeDetection Detect(const DeviceMonitorConfig& config) override
            {
                ProbeDetection det;
#ifdef __APPLE__
                det.supported = false;
                det.confidence = 0;
                det.evidence.push_back("Platform is macOS (Darwin)");
#else
                if (HasNpuSmi())
                {
                    det.supported = true;
                    det.confidence = 90;
                    det.evidence.push_back("npu-smi found in system path");
                }
                else
                {
                    det.supported = false;
                    det.confidence = 0;
                    det.evidence.push_back("No Ascend NPU signature found");
                }
#endif
                return det;
            }

            void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) override
            {
#ifndef __APPLE__
                if (HasNpuSmi() && config.enable_external_commands)
                {
                    // Update npu_model dynamically during collection
                }
#endif
            }

            void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
                // Ascend metrics are collected via expensive commands
            }

            void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) override
            {
#ifndef __APPLE__
                uint64_t now_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch()).count();

                if (HasNpuSmi() && config.enable_external_commands)
                {
                    auto res = CommandRunner::Run({"npu-smi", "info"}, config.command_timeout_ms);
                    if (res.success && !res.stdout_data.empty())
                    {
                        struct ParsedNpu
                        {
                            int id = -1;
                            std::string name;
                            double temp = -1;
                            double util = -1;
                            double mem_used = -1;
                            double mem_total = -1;
                        };

                        std::map<int, ParsedNpu> npu_map;
                        std::stringstream ss(res.stdout_data);
                        std::string line;

                        // Check if it's table format
                        bool is_table = false;
                        if (res.stdout_data.find("+--") != std::string::npos || res.stdout_data.find("| NPU") != std::string::npos)
                        {
                            is_table = true;
                        }

                        if (is_table)
                        {
                            while (std::getline(ss, line))
                            {
                                line = Trim(line);
                                if (line.empty() || line.front() != '|') continue;

                                auto cols = Split(line, '|');
                                if (cols.size() < 5) continue;

                                // Clean columns
                                for (auto& col : cols)
                                {
                                    col = Trim(col);
                                }

                                // Header line check
                                if (cols[1].find("NPU") != std::string::npos || cols[1].find("Chip") != std::string::npos)
                                    continue;

                                // Parse NPU ID and name / device name
                                std::stringstream col1_ss(cols[1]);
                                int npu_id = -1;
                                std::string item_name;
                                if (col1_ss >> npu_id >> item_name)
                                {
                                    // check if item_name contains "device"
                                    if (item_name.rfind("device", 0) == 0)
                                    {
                                        // Line 2 (Bus-Id, AICore / Chip-Util, Memory-Usage)
                                        auto& npu = npu_map[npu_id];
                                        npu.id = npu_id;
                                        
                                        // AICore / Chip-Util
                                        try
                                        {
                                            npu.util = std::stod(cols[3]);
                                        }
                                        catch (...) {}

                                        // Memory-Usage: "1205 / 8192"
                                        auto mem_parts = Split(cols[4], '/');
                                        if (mem_parts.size() >= 2)
                                        {
                                            try
                                            {
                                                npu.mem_used = std::stod(Trim(mem_parts[0]));
                                                npu.mem_total = std::stod(Trim(mem_parts[1]));
                                            }
                                            catch (...) {}
                                        }
                                    }
                                    else
                                    {
                                        // Line 1 (Name, Health, Power, Temp)
                                        auto& npu = npu_map[npu_id];
                                        npu.id = npu_id;
                                        npu.name = "Ascend " + item_name;
                                        try
                                        {
                                            npu.temp = std::stod(cols[4]);
                                        }
                                        catch (...) {}
                                    }
                                }
                            }
                        }
                        else
                        {
                            // Key-Value style fallback (e.g. from npu-smi info -t/m/etc)
                            ParsedNpu single_npu;
                            single_npu.id = 0;
                            single_npu.name = "Ascend NPU";
                            bool parsed_something = false;

                            while (std::getline(ss, line))
                            {
                                size_t colon = line.find(":");
                                if (colon != std::string::npos)
                                {
                                    std::string key = Trim(line.substr(0, colon));
                                    std::string val = Trim(line.substr(colon + 1));
                                    std::transform(key.begin(), key.end(), key.begin(), ::tolower);

                                    parsed_something = true;
                                    if (key.find("name") != std::string::npos || key.find("model") != std::string::npos)
                                    {
                                        single_npu.name = "Ascend " + val;
                                    }
                                    else if (key.find("temp") != std::string::npos)
                                    {
                                        try { single_npu.temp = std::stod(val); } catch (...) {}
                                    }
                                    else if (key.find("util") != std::string::npos || key.find("ai core") != std::string::npos)
                                    {
                                        try { single_npu.util = std::stod(val); } catch (...) {}
                                    }
                                    else if (key.find("memory") != std::string::npos)
                                    {
                                        auto mem_parts = Split(val, '/');
                                        if (mem_parts.size() >= 2)
                                        {
                                            try
                                            {
                                                single_npu.mem_used = std::stod(Trim(mem_parts[0]));
                                                single_npu.mem_total = std::stod(Trim(mem_parts[1]));
                                            }
                                            catch (...) {}
                                        }
                                    }
                                }
                            }

                            if (parsed_something)
                            {
                                npu_map[0] = single_npu;
                            }
                        }

                        if (!npu_map.empty())
                        {
                            double max_util = 0.0;
                            std::vector<AcceleratorInfo> ascend_npus;

                            for (const auto& pair : npu_map)
                            {
                                const auto& parsed = pair.second;
                                AcceleratorInfo npu;
                                npu.type = "npu";
                                npu.vendor = "ascend";
                                npu.name = parsed.name.empty() ? "Ascend NPU" : parsed.name;

                                if (parsed.util >= 0)
                                {
                                    npu.usage.available = true;
                                    npu.usage.stale = false;
                                    npu.usage.value = parsed.util;
                                    npu.usage.unit = "percent";
                                    npu.usage.source = "npu-smi";
                                    npu.usage.collected_at_ms = now_ms;
                                    max_util = std::max(max_util, parsed.util);
                                }

                                if (parsed.mem_total > 0)
                                {
                                    npu.memory_usage.available = true;
                                    npu.memory_usage.stale = false;
                                    npu.memory_usage.value = (parsed.mem_used / parsed.mem_total) * 100.0;
                                    npu.memory_usage.unit = "percent";
                                    npu.memory_usage.source = "npu-smi";
                                    npu.memory_usage.collected_at_ms = now_ms;
                                }

                                if (parsed.temp >= 0)
                                {
                                    npu.temperature.available = true;
                                    npu.temperature.stale = false;
                                    npu.temperature.value = parsed.temp;
                                    npu.temperature.unit = "celsius";
                                    npu.temperature.source = "npu-smi";
                                    npu.temperature.collected_at_ms = now_ms;
                                }

                                ascend_npus.push_back(npu);
                            }

                            metrics.npu_usage.available = true;
                            metrics.npu_usage.stale = false;
                            metrics.npu_usage.value = max_util;
                            metrics.npu_usage.unit = "percent";
                            metrics.npu_usage.source = "npu-smi (max)";
                            metrics.npu_usage.collected_at_ms = now_ms;

                            // Update the main accelerators list, removing any old ascend NPUs
                            accelerators.erase(
                                std::remove_if(accelerators.begin(), accelerators.end(), 
                                               [](const AcceleratorInfo& acc) {
                                                   return acc.vendor == "ascend";
                                               }),
                                accelerators.end()
                            );

                            for (const auto& npu : ascend_npus)
                            {
                                accelerators.push_back(npu);
                            }
                        }
                    }
                    else
                    {
                        metrics.npu_usage.available = false;
                        metrics.npu_usage.error = "npu-smi execution failed: " + res.error;
                        throw std::runtime_error("npu-smi collection failed: " + res.error);
                    }
                }
#endif
            }
        };

        std::shared_ptr<IDeviceProbe> CreateAscendProbe()
        {
            return std::make_shared<AscendProbe>();
        }
    }
}
