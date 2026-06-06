#ifndef AIVISION_PIPELINE_WORKER_POOL_H
#define AIVISION_PIPELINE_WORKER_POOL_H

// NPU Worker 线程池。
// 核心设计：
//   1. 固定数量 Worker 线程，从所有流队列轮询提取 FrameContext。
//   2. 每个 Worker 获取快照后的 AlgoConfig，在帧上串联执行算法。
//   3. 多算法串联：detector_infer 的 context_json 参数承载上一级输出。
//   4. 同一帧图像 (HwBuffer) 零拷贝传递给所有绑定算法。

#include <atomic>
#include <condition_variable>
#include <functional>
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

#include "hw_buffer.h"
#include "ring_queue.h"
#include "snapshot.h"
#include "algo/algo_manager.h"

namespace aivision
{
    namespace pipeline
    {

        /// 推理结果回调
        struct InferResult
        {
            std::string task_id;
            std::string algo_name;
            uint64_t frame_ts_ns;
            bool success = false;
            std::string result_json;
            uint32_t infer_time_us = 0;
        };

        using ResultCallback = std::function<void(const InferResult &)>;

        /// Worker 配置
        struct WorkerPoolConfig
        {
            /// Worker 线程数 (通常 = NPU 核心数)
            uint32_t worker_count = 4;

            /// 轮询间隔 (毫秒)
            uint32_t poll_interval_ms = 10;

            /// 单帧推理超时 (毫秒)
            uint32_t infer_timeout_ms = 5000;

            /// 空闲 Worker 超时后休眠 (毫秒)
            uint32_t idle_sleep_ms = 100;
        };

        /// Worker 池
        class WorkerPool
        {
        public:
            explicit WorkerPool(const WorkerPoolConfig &config);
            ~WorkerPool();

            /// 启动所有 Worker 线程
            void Start();

            /// 停止所有 Worker 线程
            void Stop();

            /// 关联流队列管理器
            void SetQueueManager(StreamQueueManager *mgr) { queue_mgr_ = mgr; }

            /// 关联快照管理器
            void SetSnapshotManager(SnapshotManager *mgr) { snapshot_mgr_ = mgr; }

            /// 关联算法管理器
            void SetAlgoManager(algo::AlgoManager *mgr) { algo_mgr_ = mgr; }

            /// 设置推理结果回调
            void SetResultCallback(ResultCallback cb) { result_cb_ = std::move(cb); }

            /// 获取 Worker 总数
            uint32_t WorkerCount() const { return config_.worker_count; }

            /// 获取空闲 Worker 数
            int32_t IdleWorkerCount() const { return idle_count_.load(); }

            /// 暂停所有 Worker (用于热更新排空)
            void PauseAll();

            /// 恢复所有 Worker
            void ResumeAll();

            /// 是否已暂停
            bool IsPaused() const { return paused_.load(); }

            /// 获取累计已处理帧数
            uint64_t TotalProcessedFrames() const { return processed_frames_.load(); }

        private:
            /// 单个 Worker 主循环
            void WorkerLoop(uint32_t worker_id);

            /// 在一帧上串联执行所有绑定的算法
            InferResult ExecuteAlgoChain(const FrameContext &frame,
                                         const std::vector<std::string> &algo_names);

            WorkerPoolConfig config_;
            std::atomic<bool> running_{false};
            std::atomic<bool> paused_{false};
            std::atomic<int32_t> idle_count_{0};
            std::atomic<uint64_t> processed_frames_{0};

            /// Worker 线程池
            std::vector<std::unique_ptr<std::thread>> workers_;

            /// 外部依赖注入
            StreamQueueManager *queue_mgr_ = nullptr;
            SnapshotManager *snapshot_mgr_ = nullptr;
            algo::AlgoManager *algo_mgr_ = nullptr;

            /// 结果回调
            ResultCallback result_cb_;

            /// 暂停控制
            std::mutex pause_mutex_;
            std::condition_variable pause_cv_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_WORKER_POOL_H
