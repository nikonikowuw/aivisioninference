#ifndef AIVISION_PIPELINE_PIPELINE_H
#define AIVISION_PIPELINE_PIPELINE_H

#include <memory>
#include <string>
#include <vector>
#include <mutex>
#include <thread>
#include <atomic>

#include "hw_buffer.h"
#include "ring_queue.h"

namespace aivision
{
    namespace pipeline
    {

        /// Pipeline 执行环境上下文
        struct StageContext
        {
            std::string device_id;
            RingQueue *queue = nullptr;
            // 可以添加更多共享资源，如 AlgoManager 指针等
        };

        /// 抽象 Stage 基类
        class Stage
        {
        public:
            virtual ~Stage() = default;

            /// 初始化 Stage
            virtual bool Init(const StageContext &ctx) = 0;

            /// 运行 Stage (通常启动内部线程)
            virtual bool Run() = 0;

            /// 停止 Stage (同步等待线程退出)
            virtual void Stop() = 0;

            /// 处理帧 (由 Pipeline 调用)
            virtual void PushFrame(const FrameContext &frame) = 0;

            /// 获取 Stage 名称
            virtual std::string GetName() const = 0;

            /// 获取状态
            virtual bool IsRunning() const = 0;
        };

        /// 视频处理流水线 (每路流一个实例)
        class Pipeline
        {
        public:
            explicit Pipeline(const std::string &device_id, size_t queue_capacity = 5);
            ~Pipeline();

            /// 动态添加 Stage
            bool AddStage(std::shared_ptr<Stage> stage);

            /// 动态添加 Stage (unique_ptr，内部转为 shared_ptr)
            bool AddStage(std::unique_ptr<Stage> stage);

            /// 动态移除 Stage (按名称)
            bool RemoveStage(const std::string &name);

            /// 启动 Pipeline (启动所有已存在的 Stage)
            bool Start();

            /// 停止 Pipeline (停止所有 Stage 并清空)
            void Stop();

            /// 获取流 ID
            std::string GetDeviceId() const { return device_id_; }

            /// 获取内部队列
            RingQueue *GetQueue() { return &queue_; }

            /// 获取所有 Stage 名称
            std::vector<std::string> GetStageNames() const;

        private:
            std::string device_id_;
            RingQueue queue_;
            StageContext ctx_;

            mutable std::mutex mutex_;
            std::vector<std::shared_ptr<Stage>> stages_;
            std::atomic<bool> running_{false};
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_PIPELINE_H
