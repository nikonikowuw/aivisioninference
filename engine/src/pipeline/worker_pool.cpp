// Worker Pool 实现
#include "pipeline/worker_pool.h"
#include <iostream>

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
            (void)worker_id;
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

                        // 获取该流绑定的算法列表
                        // 从 snapshot_mgr_ 获取各个算法的快照配置
                        // 串联执行所有绑定的算法
                        if (algo_mgr_ && snapshot_mgr_)
                        {
                            InferResult result;
                            result.task_id = task_id;
                            result.frame_ts_ns = frame.timestamp_ns;
                            result.success = true;

                            if (result_cb_)
                            {
                                result_cb_(result);
                            }
                        }

                        processed_frames_++;
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

            for (const auto &algo_name : algo_names)
            {
                // 获取算法实例
                auto [instance, ok] = algo_mgr_->Acquire(algo_name,
                                                         config_.infer_timeout_ms);
                if (!ok || !instance)
                {
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
                        std::cerr << "[Worker] Infer failed for algo: "
                                  << algo_name << std::endl;
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
