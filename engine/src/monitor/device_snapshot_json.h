#ifndef AIVISION_MONITOR_DEVICE_SNAPSHOT_JSON_H
#define AIVISION_MONITOR_DEVICE_SNAPSHOT_JSON_H

#include "monitor/device_monitor.h"
#include <string>

namespace aivision
{
    namespace monitor
    {
        /// 将 DeviceSnapshot 序列化为 Heartbeat JSON
        std::string ToHeartbeatJson(
            uint64_t uptime,
            size_t current_load,
            const std::string& engine_version,
            const std::string& hal_platform,
            const std::string& installed_algorithms_json_array_str,
            const DeviceSnapshot& snapshot);

        /// 将 DeviceSnapshot 序列化为完整的 /api/engine/device API 响应 JSON
        std::string ToEngineDeviceApiJson(
            uint64_t uptime,
            size_t current_load,
            const std::string& engine_version,
            const std::string& hal_platform,
            const DeviceSnapshot& snapshot);

        /// 将 DeviceSnapshot 序列化为旧的 /hardware-info API 响应 JSON
        std::string ToLegacyHardwareInfoJson(
            const std::string& hal_platform,
            const DeviceSnapshot& snapshot);
    }
}

#endif // AIVISION_MONITOR_DEVICE_SNAPSHOT_JSON_H
