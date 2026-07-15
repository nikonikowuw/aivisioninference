#ifndef AIVISION_MONITOR_METRICS_REPORTER_H
#define AIVISION_MONITOR_METRICS_REPORTER_H

// MetricsReporter — 后台守护线程，主动拉取并上报引擎指标。
// 核心设计：
//   1. 每 5 秒拉取 WorkerPool、AlgoManager、HwBufferPool 内部状态。
//   2. 构造 EngineMetricsMsg (FlatBuffers) 主动推送到 Go 控制面。
//   3. 为多机部署下的负载均衡调度提供数据支撑。

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>
#include <unordered_map>

#include "response_router.h"
#include "pipeline/worker_pool.h"
#include "pipeline/hw_buffer.h"
#include "pipeline/ring_queue.h"
#include "algo/algo_manager.h"
#include "pipeline/pipeline_manager.h"

namespace aivision
{
    namespace monitor
    {
        class DeviceMonitor;

        /// 引擎全局指标快照
        struct EngineMetrics
        {
            /// 各流指标
            struct StreamMetric
            {
                std::string task_id;
                uint8_t queue_depth = 0;
                uint64_t evict_count = 0;
                uint32_t last_infer_latency_us = 0;
                uint32_t avg_infer_latency_us = 0;
                uint32_t p95_infer_latency_us = 0;
                uint64_t frame_count = 0;
                uint64_t last_frame_ts_ns = 0;
            };

            std::vector<StreamMetric> streams;
            uint32_t active_stream_count = 0;
            uint64_t dma_used_bytes = 0;
            uint64_t dma_total_bytes = 0;
            uint64_t npu_used_bytes = 0;
            uint64_t npu_total_bytes = 0;
            uint32_t worker_count = 0;
            uint32_t idle_worker_count = 0;
            uint64_t timestamp_ns = 0;
            uint32_t decode_sessions = 0;
            uint32_t encode_sessions = 0;
            uint32_t decode_slots_used = 0;
            uint32_t encode_slots_used = 0;
            uint64_t egress_bps = 0;
            uint32_t preview_pipeline_count = 0;
            uint32_t inference_pipeline_count = 0;
            uint32_t mixed_pipeline_count = 0;
            bool media_metrics_valid = false;
			uint32_t preview_capacity = 0;
			uint32_t preview_in_use = 0;
			bool preview_capacity_valid = false;
            float accelerator_utilization = 0.0f;
            bool accelerator_metrics_valid = false;
        };

        /// 指标回调 (由 MetricsReporter 构造后传递)
        using MetricsCallback = std::function<void(const EngineMetrics &)>;

        /// MetricsReporter 配置
        struct MetricsReporterConfig
        {
            /// 上报间隔 (毫秒)
            uint32_t report_interval_ms = 5000;

            /// 指标采集超时 (毫秒)
            uint32_t collect_timeout_ms = 1000;
			uint32_t preview_capacity = 0;
        };

        /// MetricsReporter — 引擎指标上报器
        class MetricsReporter
        {
        public:
            MetricsReporter(ResponseRouter *response_router,
                            pipeline::WorkerPool *worker_pool,
                            pipeline::HwBufferPool *buffer_pool,
                            pipeline::StreamQueueManager *queue_mgr,
                            algo::AlgoManager *algo_mgr,
                            pipeline::PipelineManager *pipeline_mgr,
                            const MetricsReporterConfig &config = MetricsReporterConfig{});

            ~MetricsReporter();

            /// 启动上报线程
            void Start();

            /// 停止上报线程
            void Stop();

            /// 是否运行中
            bool IsRunning() const { return running_.load(); }

            /// 设置自定义指标回调 (用于单元测试或额外消费)
            void SetMetricsCallback(MetricsCallback cb) { metrics_cb_ = std::move(cb); }

            /// 关联 DeviceMonitor 以获取完整设备快照
            void SetDeviceMonitor(DeviceMonitor *monitor) { device_monitor_ = monitor; }

            /// 手动触发一次采集 (同步)
            EngineMetrics CollectNow();

        private:
            /// 上报主循环
            void ReportLoop();

            /// 采集一次指标
            EngineMetrics CollectMetrics();

            ResponseRouter *response_router_;
            pipeline::WorkerPool *worker_pool_;
            pipeline::HwBufferPool *buffer_pool_;
            pipeline::StreamQueueManager *queue_mgr_;
            algo::AlgoManager *algo_mgr_;
            pipeline::PipelineManager *pipeline_mgr_;
            DeviceMonitor *device_monitor_ = nullptr;

            MetricsReporterConfig config_;
            std::atomic<bool> running_{false};
            std::unique_ptr<std::thread> reporter_thread_;
            std::mutex stop_mutex_;
            std::condition_variable stop_cv_;

            /// 外部指标回调
            MetricsCallback metrics_cb_;
            std::mutex media_state_mutex_;
            std::unordered_map<std::string, uint64_t> previous_egress_bytes_;
            uint64_t previous_media_timestamp_ns_ = 0;
        };

    } // namespace monitor
} // namespace aivision

#endif // AIVISION_MONITOR_METRICS_REPORTER_H
