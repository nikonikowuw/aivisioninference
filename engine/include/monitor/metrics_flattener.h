#ifndef AIVISION_MONITOR_METRICS_FLATTENER_H
#define AIVISION_MONITOR_METRICS_FLATTENER_H

#include <string>
#include <vector>
#include <memory>

#include "monitor/device_monitor.h"

namespace aivision
{
    namespace monitor
    {

        /// 单挂载点磁盘使用信息
        struct DiskUsageInfo
        {
            std::string path;   // 挂载点路径
            uint64_t total = 0; // 总字节数
            uint64_t used = 0;  // 已用字节数
            double percent = 0.0; // 使用率 0-100
        };

        /// 平坦化的指标结构，直接映射到心跳 JSON 字段
        struct FlattenedMetrics
        {
            // CPU
            double cpu_usage = 0.0;       // 0-100
            double cpu_load_1m = 0.0;
            double cpu_load_5m = 0.0;
            double cpu_load_15m = 0.0;

            // Memory
            double memory_usage = 0.0;    // 0-100
            uint64_t memory_used = 0;     // bytes
            uint64_t memory_total = 0;    // bytes

            // Disk per mount point
            std::vector<DiskUsageInfo> disk_usage;

            // Network
            uint64_t net_rx_bytes = 0;
            uint64_t net_tx_bytes = 0;
            double net_rx_speed = 0.0;    // bytes/s
            double net_tx_speed = 0.0;    // bytes/s

            // System
            uint64_t uptime = 0;          // seconds
            int process_count = 0;
            int thread_count = 0;
            double temperature = 0.0;     // Celsius

            // Engine-specific
            int worker_count = 0;
            int idle_worker_count = 0;
            int active_stream_count = 0;
            int decode_sessions = 0;
            int encode_sessions = 0;
            int current_load = 0;
        };

        /// MetricsFlattener — 将 DeviceSnapshot 平坦化为心跳 JSON 可用的扁平字段
        ///
        /// 设计目标：
        /// 1. 零拷贝引用 DeviceSnapshot（通过 shared_ptr 传递）
        /// 2. 计算派生指标（如网络速率，当前值 - 上次值）/ 间隔
        /// 3. 提供纯数据转换，不处理序列化
        class MetricsFlattener
        {
        public:
            MetricsFlattener();

            /// 从 DeviceSnapshot 平坦化为 FlattenedMetrics
            /// @param snapshot 当前设备快照
            /// @param interval_ms 距离上次采集的间隔毫秒数（用于计算网络速率）
            FlattenedMetrics Flatten(const DeviceSnapshot& snapshot, uint64_t interval_ms);

            /// 从上次调用的 snapshot 中获取磁盘挂载点列表并按路径合并
            static std::vector<DiskUsageInfo> ExtractDiskUsage(const DeviceSnapshot& snapshot);

        private:
            /// 从 /proc/net/dev 解析网络字节计数
            /// LinuxGenericProbe 已经填充了网络字段，这里做后处理
            uint64_t last_net_rx_bytes_ = 0;
            uint64_t last_net_tx_bytes_ = 0;
            uint64_t last_timestamp_ms_ = 0;
        };

    } // namespace monitor
} // namespace aivision

#endif // AIVISION_MONITOR_METRICS_FLATTENER_H
