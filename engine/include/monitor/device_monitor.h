#ifndef AIVISION_MONITOR_DEVICE_MONITOR_H
#define AIVISION_MONITOR_DEVICE_MONITOR_H

#include <string>
#include <vector>
#include <mutex>
#include <memory>
#include <thread>
#include <atomic>
#include <algorithm>

namespace aivision
{
    namespace monitor
    {
        // forward declarations of internal interfaces
        class IDeviceProbe;

        /// 指标值结构，带可用性、源及错误详情
        struct MetricValue
        {
            bool available = false;
            bool stale = false;
            double value = 0.0;
            std::string unit;
            std::string source;
            std::string error;
            uint64_t collected_at_ms = 0;
        };

        /// 硬件静态配置与系统静态信息
        struct DeviceStaticInfo
        {
            std::string hostname;
            std::string os;
            std::string kernel;
            std::string arch;
            std::string board;
            std::string cpu_model;
            int cpu_cores = 0;
            std::string gpu_model;
            std::string npu_model;
            uint64_t total_memory = 0;   // 字节
            uint64_t total_storage = 0;  // 字节
        };

        /// 实时系统动态指标
        struct DeviceDynamicMetrics
        {
            MetricValue cpu_usage;
            MetricValue memory_usage;
            MetricValue storage_usage;
            MetricValue gpu_usage;
            MetricValue npu_usage;
            MetricValue temperature;
        };

        /// 硬件加速器单核心/通道指标 (主要用于 RKNPU 等多核芯片)
        struct AcceleratorCoreMetric
        {
            std::string name;
            double usage = 0.0;
        };

        /// 硬件加速器（GPU / NPU）详细指标与静态信息
        struct AcceleratorInfo
        {
            std::string type;            // "gpu" 或 "npu"
            std::string vendor;          // "nvidia", "rockchip", "ascend", "apple", etc.
            std::string name;
            MetricValue usage;
            MetricValue memory_usage;    // GPU HBM/显存使用率 (扩展字段)
            MetricValue temperature;     // 核心温度
            std::vector<AcceleratorCoreMetric> cores; // 独立核心指标
        };

        /// 探测诊断结果
        struct ProbeDiagnostic
        {
            std::string name;
            bool success = false;
            std::string error;
            std::vector<std::string> evidence;
        };

        /// 完整的设备快照
        struct DeviceSnapshot
        {
            uint64_t timestamp_ms = 0;
            DeviceStaticInfo info;
            DeviceDynamicMetrics metrics;
            std::vector<AcceleratorInfo> accelerators;
            std::vector<ProbeDiagnostic> diagnostics;
        };

        /// DeviceMonitor 运行配置
        struct DeviceMonitorConfig
        {
            std::string platform_override;          // NIKO_ENGINE_DEVICE_PLATFORM env override hint
            std::string storage_path = "/";         // NIKO_ENGINE_DEVICE_STORAGE_PATH (defaults to "/")
            std::string proc_dir = "/proc";         // procfs directory (for mocking/testing)
            std::string sys_dir = "/sys";           // sysfs directory (for mocking/testing)
            bool enable_external_commands = true;   // NIKO_ENGINE_DEVICE_ENABLE_COMMANDS (defaults to true)
            uint32_t command_timeout_ms = 1500;      // NIKO_ENGINE_DEVICE_COMMAND_TIMEOUT_MS (defaults to 1500)
            uint32_t light_probe_interval_ms = 5000; // default 5s
            uint32_t expensive_probe_interval_ms = 20000; // default 20s
        };

        /// 设备硬件和资源状态后台监控管理器
        class DeviceMonitor
        {
        public:
            explicit DeviceMonitor(const DeviceMonitorConfig& config);
            ~DeviceMonitor();

            /// 初始化探测模块，确定激活的 probe 列表
            bool Initialize();

            /// 启动后台指标更新线程
            void Start();

            /// 停止后台指标更新线程并 join
            void Stop();

            /// 获取当前快照的共享指针 (线程安全，零拷贝读取)
            std::shared_ptr<const DeviceSnapshot> GetSnapshotPtr();

            /// 获取缓存快照的深拷贝 (线程安全)
            DeviceSnapshot GetSnapshot();

        private:
            /// 后台线程循环函数
            void MonitorLoop();

            /// 发布新的快照并通知
            void PublishSnapshot(const DeviceSnapshot& new_snapshot);

            DeviceMonitorConfig config_;
            std::vector<std::shared_ptr<IDeviceProbe>> probes_;
            std::vector<std::shared_ptr<IDeviceProbe>> active_probes_;

            std::mutex mutex_;
            std::shared_ptr<const DeviceSnapshot> cached_snapshot_ptr_;

            std::atomic<bool> running_{false};
            std::unique_ptr<std::thread> thread_;
        };
    }
}

#endif // AIVISION_MONITOR_DEVICE_MONITOR_H
