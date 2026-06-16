#ifndef AIVISION_MONITOR_PROBES_DEVICE_PROBE_H
#define AIVISION_MONITOR_PROBES_DEVICE_PROBE_H

#include "monitor/device_monitor.h"
#include <string>
#include <vector>

namespace aivision
{
    namespace monitor
    {
        struct ProbeDetection
        {
            bool supported = false;
            int confidence = 0;
            std::vector<std::string> evidence;
        };

        class IDeviceProbe
        {
        public:
            virtual ~IDeviceProbe() = default;

            /// 返回 Probe 唯一标识名称
            virtual std::string Name() const = 0;

            /// 检测当前运行时环境是否支持该 Probe，返回置信度及证据
            virtual ProbeDetection Detect(const DeviceMonitorConfig& config) = 0;

            /// 收集静态硬件配置信息
            virtual void CollectStaticInfo(const DeviceMonitorConfig& config, DeviceStaticInfo& info) = 0;

            /// 收集轻量级动态指标 (应当极快，通常 < 50ms)
            virtual void CollectDynamicMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) = 0;

            /// 收集昂贵动态指标 (如外部 CLI 调用，可能需要 500ms~1500ms)
            virtual void CollectExpensiveMetrics(const DeviceMonitorConfig& config, DeviceDynamicMetrics& metrics, std::vector<AcceleratorInfo>& accelerators) = 0;
        };

        // Factory functions for concrete probes
        std::shared_ptr<IDeviceProbe> CreateLinuxGenericProbe();
        std::shared_ptr<IDeviceProbe> CreateRockchipProbe();
        std::shared_ptr<IDeviceProbe> CreateNvidiaProbe();
        std::shared_ptr<IDeviceProbe> CreateAscendProbe();
        std::shared_ptr<IDeviceProbe> CreateMacOSProbe();

        /// Command Runner structures and interface for expensive CLI calls
        struct CommandResult
        {
            bool success = false;
            int exit_code = -1;
            std::string stdout_data;
            std::string stderr_data;
            std::string error;
        };

        class CommandRunner
        {
        public:
            static CommandResult Run(const std::vector<std::string>& args, uint32_t timeout_ms, size_t max_output_bytes = 65536);
        };
    }
}

#endif // AIVISION_MONITOR_PROBES_DEVICE_PROBE_H
