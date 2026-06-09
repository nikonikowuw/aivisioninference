#ifndef AIVISION_PIPELINE_RING_QUEUE_H
#define AIVISION_PIPELINE_RING_QUEUE_H

// Ring Queue — 视频流专用有界队列。
// 核心设计：
//   1. 固定容量 (默认 5 帧，对应约 200ms 缓冲)。
//   2. push 时若队列已满，自动挤出（丢弃）最老的一帧。
//   3. 被挤出的帧释放其 HwBuffer 引用（引用计数归零时回收 DMA 内存）。
//   4. 支持预警日志：首次挤出和每 N 次挤出记录日志。

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <functional>
#include <memory>
#include <mutex>
#include <queue>
#include <string>
#include <unordered_map>

#include "hw_buffer.h"

namespace aivision
{
    namespace pipeline
    {

        /// 帧上下文：包含图像数据和算法串联所需的累积 JSON
        struct FrameContext
        {
            /// 图像缓冲 (智能指针，零拷贝)
            HwBufferPtr buffer;

            /// 帧时间戳 (Unix 纳秒)
            uint64_t timestamp_ns = 0;

            /// 帧序号 (单调递增)
            uint64_t frame_seq = 0;

            /// 流任务 ID
            std::string task_id;

            /// 算法串联累积 JSON 结果。
            /// Worker 每执行完一个算法，将结果附加到此 JSON，
            /// 下一个算法从同一 FrameContext 读取作为输入。
            std::string accumulated_json;
        };

        /// Ring Queue 类
        class RingQueue
        {
        public:
            explicit RingQueue(size_t capacity = 5);

            /// 推入一帧。若队列已满，自动挤出最老帧。
            /// 返回被挤出的帧（可为空，表示队列未满）。
            /// 线程安全。
            FrameContext Push(FrameContext frame);

            /// 尝试弹出最老的帧。如果队列为空，阻塞等待最多 timeout_ms。
            /// 返回帧和是否成功。
            std::pair<FrameContext, bool> TryPop(uint32_t timeout_ms = 100);

            /// 非阻塞弹出
            FrameContext TryPopNonBlock();

            /// 获取当前队列深度
            size_t Size() const;

            /// 检查队列是否为空
            bool IsEmpty() const;

            /// 清空队列 (丢弃所有帧)
            void Clear();

            /// 获取容量
            size_t Capacity() const { return capacity_; }

            /// 获取累计挤出帧计数
            uint64_t EvictCount() const { return evict_count_.load(); }

            /// 设置挤出事件回调 (用于日志和 Metrics)
            using EvictCallback = std::function<void(const FrameContext &)>;
            void SetEvictCallback(EvictCallback cb) { evict_cb_ = std::move(cb); }

            /// 设置预警日志间隔 (默认每 100 次挤出记录一次)
            void SetLogInterval(size_t interval) { log_interval_ = interval; }

        private:
            /// 内部环形缓冲
            size_t capacity_;
            std::queue<FrameContext> queue_;
            mutable std::mutex mutex_;
            std::condition_variable cv_;

            /// 统计
            std::atomic<uint64_t> evict_count_{0};

            /// 回调
            EvictCallback evict_cb_;

            /// 日志间隔
            size_t log_interval_ = 100;

            /// 首次挤出标记
            bool first_evict_logged_ = false;
        };

        /// 流队列管理器 — 管理所有流的 RingQueue
        class StreamQueueManager
        {
        public:
            StreamQueueManager() = default;
            ~StreamQueueManager() = default;

            /// 为指定流创建队列
            RingQueue *CreateStream(const std::string &task_id);

            /// 获取指定流的队列
            RingQueue *GetStream(const std::string &task_id);

            /// 移除指定流的队列
            void RemoveStream(const std::string &task_id);

            /// 是否有活跃流
            bool HasActiveStreams() const;

            /// 获取活跃流数
            size_t ActiveStreamCount() const;

            /// 获取所有流 ID
            std::vector<std::string> GetAllStreamIds() const;

        private:
            mutable std::mutex mutex_;
            std::unordered_map<std::string, std::unique_ptr<RingQueue>> streams_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_RING_QUEUE_H
