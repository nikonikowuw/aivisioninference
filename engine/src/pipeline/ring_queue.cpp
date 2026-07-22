// Ring Queue 实现
#include "pipeline/ring_queue.h"
#include "logger/logger.h"

namespace aivision
{
    namespace pipeline
    {

        RingQueue::RingQueue(size_t capacity)
            : capacity_(capacity) {}

        FrameContext RingQueue::Push(FrameContext frame)
        {
            FrameContext evicted;

            std::lock_guard<std::mutex> lock(mutex_);

            if (queue_.size() >= capacity_)
            {
                // 挤出最老的一帧
                evicted = std::move(queue_.front());
                queue_.pop();
                evict_count_++;

                // 预警日志
                if (!first_evict_logged_)
                {
                    LOG_WARN("[RingQueue] First frame evicted for capacity={}", capacity_);
                    first_evict_logged_ = true;
                }
                if (evict_count_.load() % log_interval_ == 0)
                {
                    LOG_WARN("[RingQueue] Evicted {} frames so far", evict_count_.load());
                }

                // 回调通知
                if (evict_cb_)
                {
                    evict_cb_(evicted);
                }
            }

            queue_.push(std::move(frame));
            cv_.notify_one();

            return evicted;
        }

        std::pair<FrameContext, bool> RingQueue::TryPop(uint32_t timeout_ms)
        {
            std::unique_lock<std::mutex> lock(mutex_);

            if (queue_.empty())
            {
                if (timeout_ms > 0)
                {
                    cv_.wait_for(lock, std::chrono::milliseconds(timeout_ms));
                }
                if (queue_.empty())
                {
                    return {FrameContext{}, false};
                }
            }

            FrameContext frame = std::move(queue_.front());
            queue_.pop();
            return {std::move(frame), true};
        }

        FrameContext RingQueue::TryPopNonBlock()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (queue_.empty())
            {
                return FrameContext{};
            }
            FrameContext frame = std::move(queue_.front());
            queue_.pop();
            return frame;
        }

        size_t RingQueue::Size() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return queue_.size();
        }

        bool RingQueue::IsEmpty() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return queue_.empty();
        }

        void RingQueue::Clear()
        {
            std::lock_guard<std::mutex> lock(mutex_);
            while (!queue_.empty())
            {
                queue_.pop();
            }
        }

        // StreamQueueManager 实现
        RingQueue *StreamQueueManager::CreateStream(const std::string &task_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto queue = std::make_unique<RingQueue>(5);
            RingQueue *ptr = queue.get();
            streams_[task_id] = std::move(queue);
            return ptr;
        }

        RingQueue *StreamQueueManager::GetStream(const std::string &task_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = streams_.find(task_id);
            return it != streams_.end() ? it->second.get() : nullptr;
        }

        void StreamQueueManager::RemoveStream(const std::string &task_id)
        {
            std::lock_guard<std::mutex> lock(mutex_);
            streams_.erase(task_id);
        }

        bool StreamQueueManager::HasActiveStreams() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return !streams_.empty();
        }

        size_t StreamQueueManager::ActiveStreamCount() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            return streams_.size();
        }

        std::vector<std::string> StreamQueueManager::GetAllStreamIds() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            std::vector<std::string> ids;
            ids.reserve(streams_.size());
            for (const auto &[id, _] : streams_)
            {
                ids.push_back(id);
            }
            return ids;
        }

    } // namespace pipeline
} // namespace aivision
