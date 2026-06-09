// Worker Pool 实现
#include "pipeline/worker_pool.h"
#include <iostream>
#include <mutex>

#ifdef __linux__
#include <pthread.h>
#endif

namespace
{
    std::mutex g_worker_log_mutex;
}

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
            std::cout << "[WorkerPool] starting workers count=" << config_.worker_count << std::endl;
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
                std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                std::cout << "[WorkerPool] worker started id=" << worker_id << std::endl;
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
                            std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                            std::cout << "[WorkerPool] frame dequeued"
                                      << " worker=" << worker_id
                                      << " stream=" << task_id
                                      << " total=" << current_frame
                                      << " queue_size=" << queue->Size()
                                      << std::endl;
                        }

                        if (algo_mgr_ && snapshot_mgr_)
                        {
                            auto algo_names = snapshot_mgr_->GetStreamAlgos(task_id);
                            if (algo_names.empty())
                            {
                                std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                                std::cerr << "[Worker] No algorithms bound for stream: "
                                          << task_id << std::endl;
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

            {
                std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                std::cout << "[Worker] executing algo chain"
                          << " stream=" << frame.task_id
                          << " algos=" << algo_names.size()
                          << " frame_ts_ns=" << frame.timestamp_ns
                          << std::endl;
            }

            for (const auto &algo_name : algo_names)
            {
                // 获取算法实例
                auto [instance, ok] = algo_mgr_->Acquire(algo_name,
                                                         config_.infer_timeout_ms);
                if (!ok || !instance)
                {
                    std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                    std::cerr << "[Worker] Failed to acquire algo: "
                              << algo_name << std::endl;
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
                        std::lock_guard<std::mutex> lock(g_worker_log_mutex);
                        std::cerr << "[Worker] Infer failed for algo: "
                                  << algo_name
                                  << " stream=" << frame.task_id
                                  << " width=" << desc.width
                                  << " height=" << desc.height
                                  << " size=" << desc.size
                                  << " dma_fd=" << desc.dma_fd
                                  << " data=" << desc.data
                                  << " stride=" << desc.stride
                                  << " pixel_format=" << desc.pixel_format
                                  << std::endl;
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
