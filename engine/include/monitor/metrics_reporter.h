#ifndef AIVISION_MONITOR_METRICS_REPORTER_H
#define AIVISION_MONITOR_METRICS_REPORTER_H

// MetricsReporter — 后台守护线程，主动拉取并上报引擎指标。
// 核心设计：
//   1. 每 5 秒拉取 WorkerPool、AlgoManager、HwBufferPool 内部状态。
//   2. 构造 EngineMetricsMsg (FlatBuffers) 主动推送到 Go 控制面。
//   3. 为多机部署下的负载均衡调度提供数据支撑。

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <thread>

#include "ipc/ipc_server.h"
#include "pipeline/worker_pool.h"
#include "pipeline/hw_buffer.h"
#include "pipeline/ring_queue.h"
#include "algo/algo_manager.h"

namespace aivision
{
    namespace monitor
    {

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
        };

        /// MetricsReporter — 引擎指标上报器
        class MetricsReporter
        {
        public:
            MetricsReporter(ipc::IPCServer *ipc_server,
                            pipeline::WorkerPool *worker_pool,
                            pipeline::HwBufferPool *buffer_pool,
                            pipeline::StreamQueueManager *queue_mgr,
                            algo::AlgoManager *algo_mgr,
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

            /// 手动触发一次采集 (同步)
            EngineMetrics CollectNow();

        private:
            /// 上报主循环
            void ReportLoop();

            /// 采集一次指标
            EngineMetrics CollectMetrics();

            /// 计算 P95 延迟
            static uint32_t CalculateP95(const std::vector<uint32_t> &latencies);

            ipc::IPCServer *ipc_server_;
            pipeline::WorkerPool *worker_pool_;
            pipeline::HwBufferPool *buffer_pool_;
            pipeline::StreamQueueManager *queue_mgr_;
            algo::AlgoManager *algo_mgr_;

            MetricsReporterConfig config_;
            std::atomic<bool> running_{false};
            std::unique_ptr<std::thread> reporter_thread_;

            /// 外部指标回调
            MetricsCallback metrics_cb_;
        };

    } // namespace monitor
} // namespace aivision

#endif // AIVISION_MONITOR_METRICS_REPORTER_H
