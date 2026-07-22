// Worker Pool 实现
#include "pipeline/worker_pool.h"
#include "logger/logger.h"
#include <mutex>

#ifdef __linux__
#include <pthread.h>
#endif

namespace aivision
{
    namespace pipeline
    {

        WorkerPool::WorkerPool(const WorkerPoolConfig &config)
            : config_(config) {}

        WorkerPool::~WorkerPool() { Stop(); }

        void WorkerPool::Start()
        {
            running_.store(true);
            idle_count_.store(static_cast<int32_t>(config_.worker_count));
            workers_.reserve(config_.worker_count);
            LOG_INFO("[WorkerPool] starting workers count={}", config_.worker_count);
            for (uint32_t i = 0; i < config_.worker_count; ++i)
            {
                workers_.emplace_back(
                    std::make_unique<std::thread>(&WorkerPool::WorkerLoop, this, i));
            }
        }

        void WorkerPool::Stop()
        {
            running_.store(false);
            // 唤醒所有等待的 Worker
            pause_cv_.notify_all();
            for (auto &w : workers_)
            {
                if (w && w->joinable())
                {
                    w->join();
                }
            }
            workers_.clear();
        }

        void WorkerPool::PauseAll()
        {
            paused_.store(true);
        }

        void WorkerPool::ResumeAll()
        {
            paused_.store(false);
            pause_cv_.notify_all();
        }

        void WorkerPool::WorkerLoop(uint32_t worker_id)
        {
#ifdef __linux__
            std::string thread_name = "infer-w" + std::to_string(worker_id);
            pthread_setname_np(pthread_self(), thread_name.substr(0, 15).c_str());
#endif
            {
                LOG_INFO("[WorkerPool] worker started id={}", worker_id);
            }
            while (running_.load())
            {
                // 检查是否暂停 (热更新排空)
                if (paused_.load())
                {
                    std::unique_lock<std::mutex> lock(pause_mutex_);
                    pause_cv_.wait(lock, [this]()
                                   { return !paused_.load() || !running_.load(); });
                    if (!running_.load())
                        break;
                }

                // 轮询所有流队列
                bool found_work = false;
                if (queue_mgr_)
                {
                    auto stream_ids = queue_mgr_->GetAllStreamIds();
                    for (const auto &task_id : stream_ids)
                    {
                        auto *queue = queue_mgr_->GetStream(task_id);
                        if (!queue)
                            continue;

                        auto frame = queue->TryPopNonBlock();
                        if (!frame.buffer)
                            continue;

                        // 找到工作，标记非空闲
                        found_work = true;
                        uint64_t current_frame = processed_frames_.fetch_add(1) + 1;
                        if (current_frame == 1 || current_frame % 100 == 0)
                        {
                            LOG_INFO("[WorkerPool] frame dequeued worker={} stream={} total={} queue_size={}",
                                     worker_id, task_id, current_frame, queue->Size());
                        }

                        if (algo_mgr_ && snapshot_mgr_)
                        {
                            auto algo_names = snapshot_mgr_->GetStreamAlgos(task_id);
                            if (algo_names.empty())
                            {
                                LOG_WARN("[Worker] No algorithms bound for stream: {}", task_id);
                            }
                            else
                            {
                                InferResult result = ExecuteAlgoChain(frame, algo_names);
                                if (result_cb_)
                                {
                                    result_cb_(result);
                                }
                            }
                        }

                        break;
                    }
                }

                if (!found_work)
                {
                    std::this_thread::sleep_for(
                        std::chrono::milliseconds(config_.idle_sleep_ms));
                }
            }
        }

        InferResult WorkerPool::ExecuteAlgoChain(
            const FrameContext &frame,
            const std::vector<std::string> &algo_names)
        {
            InferResult result;
            result.task_id = frame.task_id;
            result.frame_ts_ns = frame.timestamp_ns;

            std::string context_json = frame.accumulated_json;

            static std::atomic<uint64_t> exec_seq{0};
            uint64_t seq = exec_seq.fetch_add(1);
            if (seq % 100 == 0) {
                LOG_INFO("[Worker] executing algo chain stream={} algos={} frame_ts_ns={}",
                         frame.task_id, algo_names.size(), frame.timestamp_ns);
            }

            for (const auto &algo_name : algo_names)
            {
                // 获取算法实例
                auto [instance, ok] = algo_mgr_->Acquire(algo_name,
                                                         config_.infer_timeout_ms);
                if (!ok || !instance)
                {
                    LOG_ERROR("[Worker] Failed to acquire algo: {}", algo_name);
                    result.success = false;
                    break;
                }

                // 推理
                if (frame.buffer && frame.buffer->IsValid())
                {
                    bool infer_ok = instance->Infer(
                        frame.buffer->Desc(),
                        context_json,
                        context_json,
                        result.infer_time_us);

                    if (!infer_ok)
                    {
                        const auto &desc = frame.buffer->Desc();
                        LOG_ERROR("[Worker] Infer failed for algo: {} stream={} width={} height={} size={} dma_fd={} data={} stride={} pixel_format={}",
                                 algo_name, frame.task_id, desc.width, desc.height, desc.size,
                                 desc.dma_fd, desc.data, desc.stride, desc.pixel_format);
                        result.success = false;
                    }
                }

                // 释放算法实例
                algo_mgr_->Release(algo_name);
            }

            result.result_json = context_json;
            return result;
        }

    } // namespace pipeline
} // namespace aivision
