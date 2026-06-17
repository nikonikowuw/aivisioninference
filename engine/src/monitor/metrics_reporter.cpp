// MetricsReporter 实现
#include "monitor/metrics_reporter.h"

#include <iostream>

namespace aivision
{
    namespace monitor
    {

        MetricsReporter::MetricsReporter(ipc::ResponseRouter *response_router,
                                         pipeline::WorkerPool *worker_pool,
                                         pipeline::HwBufferPool *buffer_pool,
                                         pipeline::StreamQueueManager *queue_mgr,
                                         algo::AlgoManager *algo_mgr,
                                         const MetricsReporterConfig &config)
            : response_router_(response_router), worker_pool_(worker_pool), buffer_pool_(buffer_pool), queue_mgr_(queue_mgr), algo_mgr_(algo_mgr), config_(config) {}

        MetricsReporter::~MetricsReporter() { Stop(); }

        void MetricsReporter::Start()
        {
            running_.store(true);
            reporter_thread_ = std::make_unique<std::thread>(
                &MetricsReporter::ReportLoop, this);
        }

        void MetricsReporter::Stop()
        {
            running_.store(false);
            stop_cv_.notify_all();
            if (reporter_thread_ && reporter_thread_->joinable())
            {
                reporter_thread_->join();
            }
        }

        void MetricsReporter::ReportLoop()
        {
            uint32_t gc_counter = 0;
            constexpr uint32_t GC_INTERVAL_CYCLES = 10; // Run GC every 10 metrics cycles

            while (running_.load())
            {
                std::unique_lock<std::mutex> lock(stop_mutex_);
                if (stop_cv_.wait_for(
                        lock,
                        std::chrono::milliseconds(config_.report_interval_ms),
                        [this]() { return !running_.load(); }))
                {
                    break;
                }

                // Run GC less frequently than metrics collection
                if (algo_mgr_ && (++gc_counter % GC_INTERVAL_CYCLES == 0))
                {
                    algo_mgr_->GarbageCollect(300000); // 5 minutes idle timeout
                }

                auto metrics = CollectMetrics();

                // 触发外部回调 (如有注册)
                if (metrics_cb_)
                {
                    metrics_cb_(metrics);
                }
            }
        }

        EngineMetrics MetricsReporter::CollectMetrics()
        {
            EngineMetrics metrics;
            metrics.timestamp_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                       std::chrono::system_clock::now().time_since_epoch())
                                       .count();

            // 采集 Worker 池状态
            if (worker_pool_)
            {
                metrics.worker_count = worker_pool_->WorkerCount();
                metrics.idle_worker_count = worker_pool_->IdleWorkerCount();
            }

            // 采集缓冲池状态
            if (buffer_pool_)
            {
                metrics.dma_used_bytes = buffer_pool_->UsedBytes();
                metrics.dma_total_bytes = buffer_pool_->TotalBytes();
            }

            // 采集各流队列状态
            if (queue_mgr_)
            {
                metrics.active_stream_count =
                    static_cast<uint32_t>(queue_mgr_->ActiveStreamCount());

                auto stream_ids = queue_mgr_->GetAllStreamIds();
                for (const auto &task_id : stream_ids)
                {
                    auto *queue = queue_mgr_->GetStream(task_id);
                    if (!queue)
                        continue;

                    EngineMetrics::StreamMetric sm;
                    sm.task_id = task_id;
                    sm.queue_depth = static_cast<uint8_t>(queue->Size());
                    sm.evict_count = queue->EvictCount();
                    metrics.streams.push_back(sm);
                }
            }

            // TODO: 采集 AlgoManager 状态 (NPU 内存使用等)
            (void)algo_mgr_;

            return metrics;
        }

        EngineMetrics MetricsReporter::CollectNow()
        {
            return CollectMetrics();
        }

    } // namespace monitor
} // namespace aivision
