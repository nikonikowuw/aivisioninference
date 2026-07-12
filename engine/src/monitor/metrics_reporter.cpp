// MetricsReporter 实现
#include "monitor/metrics_reporter.h"

#include <iostream>

namespace aivision
{
    namespace monitor
    {

        MetricsReporter::MetricsReporter(ResponseRouter *response_router,
                                         pipeline::WorkerPool *worker_pool,
                                         pipeline::HwBufferPool *buffer_pool,
                                         pipeline::StreamQueueManager *queue_mgr,
                                         algo::AlgoManager *algo_mgr,
                                         pipeline::PipelineManager *pipeline_mgr,
                                         const MetricsReporterConfig &config)
            : response_router_(response_router), worker_pool_(worker_pool), buffer_pool_(buffer_pool), queue_mgr_(queue_mgr), algo_mgr_(algo_mgr), pipeline_mgr_(pipeline_mgr), config_(config) {}

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

            if (pipeline_mgr_)
            {
                const auto pipelines = pipeline_mgr_->ListPipelines();
                bool observable = true;
                bool needs_egress_baseline = false;
                uint64_t interval_bytes = 0;
                std::unordered_map<std::string, uint64_t> current_bytes;

                std::lock_guard<std::mutex> lock(media_state_mutex_);
                for (const auto &pipeline : pipelines)
                {
                    if (pipeline.uses_decoder)
                    {
                        ++metrics.decode_sessions;
                        ++metrics.decode_slots_used;
                    }
                    if (pipeline.uses_encoder)
                    {
                        ++metrics.encode_sessions;
                        ++metrics.encode_slots_used;
                    }
                    if (pipeline.inference_enabled && pipeline.playback_enabled)
                        ++metrics.mixed_pipeline_count;
                    else if (pipeline.inference_enabled)
                        ++metrics.inference_pipeline_count;
                    else if (pipeline.playback_enabled)
                        ++metrics.preview_pipeline_count;

                    if (!pipeline.playback_enabled)
                        continue;
                    if (!pipeline.egress_observable)
                    {
                        observable = false;
                        continue;
                    }
                    current_bytes[pipeline.device_id] = pipeline.total_egress_bytes;
                    auto previous = previous_egress_bytes_.find(pipeline.device_id);
                    if (previous == previous_egress_bytes_.end() || pipeline.total_egress_bytes < previous->second)
                    {
                        needs_egress_baseline = true;
                        continue;
                    }
                    interval_bytes += pipeline.total_egress_bytes - previous->second;
                }

                if (previous_media_timestamp_ns_ > 0 && metrics.timestamp_ns > previous_media_timestamp_ns_)
                {
                    const uint64_t elapsed_ns = metrics.timestamp_ns - previous_media_timestamp_ns_;
                    metrics.egress_bps = static_cast<uint64_t>(
                        (static_cast<long double>(interval_bytes) * 8.0L * 1000000000.0L) /
                        static_cast<long double>(elapsed_ns));
                }
                else if (!current_bytes.empty())
                {
                    needs_egress_baseline = true;
                }

                metrics.media_metrics_valid = observable && !needs_egress_baseline;
                previous_egress_bytes_ = std::move(current_bytes);
                previous_media_timestamp_ns_ = metrics.timestamp_ns;
            }

            return metrics;
        }

        EngineMetrics MetricsReporter::CollectNow()
        {
            return CollectMetrics();
        }

    } // namespace monitor
} // namespace aivision
