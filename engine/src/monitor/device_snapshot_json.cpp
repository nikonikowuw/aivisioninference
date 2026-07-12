#include "monitor/device_snapshot_json.h"
#include <nlohmann/json.hpp>
#include <iostream>

namespace aivision
{
    namespace monitor
    {
        using json = nlohmann::json;

        namespace
        {
            json MetricValueToJson(const MetricValue& metric)
            {
                json j;
                j["available"] = metric.available;
                if (metric.stale)
                {
                    j["stale"] = true;
                }
                if (metric.available)
                {
                    j["value"] = metric.value;
                }
                else if (!metric.error.empty())
                {
                    j["error"] = metric.error;
                }
                j["unit"] = metric.unit;
                j["source"] = metric.source;
                return j;
            }

            json DeviceStaticInfoToJson(const DeviceStaticInfo& info)
            {
                json j;
                j["hostname"] = info.hostname;
                j["os"] = info.os;
                j["kernel"] = info.kernel;
                j["arch"] = info.arch;
                j["board"] = info.board;
                j["cpu_model"] = info.cpu_model;
                j["cpu_cores"] = info.cpu_cores;
                j["gpu_model"] = info.gpu_model;
                j["npu_model"] = info.npu_model;
                j["total_memory"] = info.total_memory;
                j["total_storage"] = info.total_storage;
                return j;
            }

            json DeviceDynamicMetricsToJson(const DeviceDynamicMetrics& metrics)
            {
                json j;
                j["cpu_usage"] = MetricValueToJson(metrics.cpu_usage);
                j["memory_usage"] = MetricValueToJson(metrics.memory_usage);
                j["storage_usage"] = MetricValueToJson(metrics.storage_usage);
                j["gpu_usage"] = MetricValueToJson(metrics.gpu_usage);
                j["npu_usage"] = MetricValueToJson(metrics.npu_usage);
                j["temperature"] = MetricValueToJson(metrics.temperature);
                // Extended metrics
                j["load_average_1m"] = MetricValueToJson(metrics.load_average_1m);
                j["load_average_5m"] = MetricValueToJson(metrics.load_average_5m);
                j["load_average_15m"] = MetricValueToJson(metrics.load_average_15m);
                j["process_count"] = MetricValueToJson(metrics.process_count);
                j["thread_count"] = MetricValueToJson(metrics.thread_count);
                return j;
            }

            json DisksToJson(const std::vector<DiskInfo>& disks)
            {
                json arr = json::array();
                for (const auto& disk : disks)
                {
                    json d;
                    d["path"] = disk.path;
                    d["total_bytes"] = disk.total_bytes;
                    d["used_bytes"] = disk.used_bytes;
                    d["usage_percent"] = disk.usage_percent;
                    arr.push_back(d);
                }
                return arr;
            }

            json NetworkStatsToJson(const NetworkStats& net)
            {
                json j;
                j["rx_bytes"] = net.rx_bytes;
                j["tx_bytes"] = net.tx_bytes;
                j["rx_speed"] = net.rx_speed;
                j["tx_speed"] = net.tx_speed;
                return j;
            }

            json AcceleratorsToJson(const std::vector<AcceleratorInfo>& accelerators)
            {
                json arr = json::array();
                for (const auto& acc : accelerators)
                {
                    json a;
                    a["type"] = acc.type;
                    a["vendor"] = acc.vendor;
                    a["name"] = acc.name;
                    a["usage"] = MetricValueToJson(acc.usage);
                    
                    if (acc.memory_usage.available)
                    {
                        a["memory_usage"] = MetricValueToJson(acc.memory_usage);
                    }
                    if (acc.temperature.available)
                    {
                        a["temperature"] = MetricValueToJson(acc.temperature);
                    }

                    if (!acc.cores.empty())
                    {
                        json cores_arr = json::array();
                        for (const auto& core : acc.cores)
                        {
                            json c;
                            c["name"] = core.name;
                            c["usage"] = core.usage;
                            cores_arr.push_back(c);
                        }
                        a["cores"] = cores_arr;
                    }
                    arr.push_back(a);
                }
                return arr;
            }
        } // namespace

        std::string ToHeartbeatJson(
            uint64_t uptime,
            size_t current_load,
            const std::string& engine_version,
            const std::string& hal_platform,
            const std::string& installed_algorithms_json_array_str,
            const DeviceSnapshot& snapshot)
        {
            json j;
            j["uptime"] = uptime;
            j["current_load"] = current_load;
            j["cpu_usage"] = snapshot.metrics.cpu_usage.available ? snapshot.metrics.cpu_usage.value : 0.0;
            j["memory_usage"] = snapshot.metrics.memory_usage.available ? snapshot.metrics.memory_usage.value : 0.0;
            j["engine_version"] = engine_version;
            j["hal_platform"] = hal_platform;

            // Legacy hardware_info
            json hw;
            hw["cpu_model"] = snapshot.info.cpu_model;
            hw["gpu_model"] = snapshot.info.gpu_model;
            hw["total_memory"] = snapshot.info.total_memory;
            hw["cpu_cores"] = snapshot.info.cpu_cores;
            j["hardware_info"] = hw;

            // Installed algorithms (parsed from Go control plane format list string)
            try
            {
                if (!installed_algorithms_json_array_str.empty() && installed_algorithms_json_array_str != "[]")
                {
                    j["installed_algorithms"] = json::parse(installed_algorithms_json_array_str);
                }
                else
                {
                    j["installed_algorithms"] = json::array();
                }
            }
            catch (...)
            {
                j["installed_algorithms"] = json::array();
            }

            // Extended flat fields for heartbeat payload
            j["cpu_load_1m"] = snapshot.metrics.load_average_1m.available ? snapshot.metrics.load_average_1m.value : 0.0;
            j["cpu_load_5m"] = snapshot.metrics.load_average_5m.available ? snapshot.metrics.load_average_5m.value : 0.0;
            j["cpu_load_15m"] = snapshot.metrics.load_average_15m.available ? snapshot.metrics.load_average_15m.value : 0.0;
            j["disks"] = DisksToJson(snapshot.disks);
            j["net_rx_bytes"] = snapshot.network.rx_bytes;
            j["net_tx_bytes"] = snapshot.network.tx_bytes;
            j["net_rx_speed"] = snapshot.network.rx_speed;
            j["net_tx_speed"] = snapshot.network.tx_speed;
            j["process_count"] = snapshot.metrics.process_count.available ? static_cast<int64_t>(snapshot.metrics.process_count.value) : 0;
            j["thread_count"] = snapshot.metrics.thread_count.available ? static_cast<int64_t>(snapshot.metrics.thread_count.value) : 0;
            j["temperature"] = snapshot.metrics.temperature.available ? snapshot.metrics.temperature.value : 0.0;

            // New fields
            j["device_info"] = DeviceStaticInfoToJson(snapshot.info);
            j["device_metrics"] = DeviceDynamicMetricsToJson(snapshot.metrics);
            j["accelerators"] = AcceleratorsToJson(snapshot.accelerators);

            // Diagnostics summary (only active probes and warnings)
            json diag;
            json active_arr = json::array();
            json warnings_arr = json::array();
            for (const auto& d : snapshot.diagnostics)
            {
                active_arr.push_back(d.name);
                if (!d.success || !d.error.empty())
                {
                    warnings_arr.push_back(d.name + ": " + d.error);
                }
            }
            diag["active_probes"] = active_arr;
            if (!warnings_arr.empty())
            {
                diag["warnings"] = warnings_arr;
            }
            j["probe_diagnostics"] = diag;

            return j.dump();
        }

        std::string ToEngineDeviceApiJson(
            uint64_t uptime,
            size_t current_load,
            const std::string& engine_version,
            const std::string& hal_platform,
            const DeviceSnapshot& snapshot)
        {
            json j;
            json runtime;
            runtime["status"] = "ok";
            runtime["uptime"] = uptime;
            runtime["current_load"] = current_load;
            runtime["engine_version"] = engine_version;
            runtime["hal_platform"] = hal_platform;
            j["runtime"] = runtime;

            json device;
            device["timestamp_ms"] = snapshot.timestamp_ms;
            device["info"] = DeviceStaticInfoToJson(snapshot.info);
            device["metrics"] = DeviceDynamicMetricsToJson(snapshot.metrics);
            device["accelerators"] = AcceleratorsToJson(snapshot.accelerators);
            device["disks"] = DisksToJson(snapshot.disks);
            device["network"] = NetworkStatsToJson(snapshot.network);

            // Diagnostics detail
            json diag_arr = json::array();
            for (const auto& d : snapshot.diagnostics)
            {
                json diag;
                diag["name"] = d.name;
                diag["success"] = d.success;
                diag["error"] = d.error;
                diag["evidence"] = d.evidence;
                diag_arr.push_back(diag);
            }
            device["diagnostics"] = diag_arr;
            j["device"] = device;

            return j.dump();
        }

        std::string ToLegacyHardwareInfoJson(
            const std::string& hal_platform,
            const DeviceSnapshot& snapshot)
        {
            json j;
            j["hal_platform"] = hal_platform;
            j["cpu_model"] = snapshot.info.cpu_model;
            j["cpu_cores"] = snapshot.info.cpu_cores;
            j["gpu_model"] = snapshot.info.gpu_model;
            j["npu_model"] = snapshot.info.npu_model;
            j["total_memory"] = snapshot.info.total_memory;
            j["total_storage"] = snapshot.info.total_storage;
            j["board"] = snapshot.info.board;
            j["arch"] = snapshot.info.arch;
            j["os"] = snapshot.info.os;
            return j.dump();
        }
    }
}
