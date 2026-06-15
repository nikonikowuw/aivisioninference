#ifndef AIVISION_IPC_HEARTBEAT_H
#define AIVISION_IPC_HEARTBEAT_H

// 心跳管理器：处理 Go 侧的心跳请求并回复 HeartbeatAck。
// 同时监控心跳超时，超时后触发重连或告警。

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <thread>

namespace aivision
{
    namespace ipc
    {

        class ResponseRouter;

        /// 心跳管理器
        class HeartbeatManager
        {
        public:
            using TimeoutCallback = std::function<void()>;

            HeartbeatManager()
                : start_time_(std::chrono::steady_clock::now()),
                  last_heartbeat_(std::chrono::steady_clock::now()) {}

            /// 构造并绑定 ResponseRouter
            explicit HeartbeatManager(ResponseRouter *router)
                : router_(router),
                  start_time_(std::chrono::steady_clock::now()),
                  last_heartbeat_(std::chrono::steady_clock::now()) {}

            ~HeartbeatManager() { Stop(); }

            /// 启动心跳监控线程
            void Start() {
                if (running_.load()) return;
                running_.store(true);
                monitor_thread_ = std::make_unique<std::thread>(&HeartbeatManager::MonitorLoop, this);
            }

            /// 停止心跳监控
            void Stop() {
                if (!running_.load()) return;
                running_.store(false);
                if (monitor_thread_ && monitor_thread_->joinable()) {
                    monitor_thread_->join();
                }
            }

            /// 刷新心跳计时器 (收到 Go 心跳时调用)
            void Refresh() {
                last_heartbeat_ = std::chrono::steady_clock::now();
                sequence_++;
            }

            /// 设置心跳超时回调
            void SetTimeoutCallback(TimeoutCallback cb) {
                timeout_cb_ = cb;
            }

            /// 获取引擎运行时长 (秒)
            uint64_t GetUptimeSec() const {
                auto now = std::chrono::steady_clock::now();
                return std::chrono::duration_cast<std::chrono::seconds>(now - start_time_).count();
            }

            /// 获取心跳序列号
            uint64_t GetSequence() const { return sequence_.load(); }

        private:
            /// 心跳监控循环
            void MonitorLoop() {
                while (running_.load()) {
                    std::this_thread::sleep_for(std::chrono::seconds(1));
                    auto now = std::chrono::steady_clock::now();
                    if (std::chrono::duration_cast<std::chrono::seconds>(now - last_heartbeat_).count() > timeout_sec_) {
                        if (timeout_cb_) {
                            timeout_cb_();
                        }
                    }
                }
            }

            ResponseRouter *router_{nullptr};
            std::atomic<bool> running_{false};
            std::atomic<uint64_t> sequence_{0};
            std::chrono::steady_clock::time_point start_time_;
            std::chrono::steady_clock::time_point last_heartbeat_;
            int timeout_sec_{30};
            std::unique_ptr<std::thread> monitor_thread_;
            TimeoutCallback timeout_cb_;

        public:
            /// 绑定 ResponseRouter (延迟绑定)
            void SetServer(ResponseRouter *router) { router_ = router; }
        };

    } // namespace ipc
} // namespace aivision

#endif // AIVISION_IPC_HEARTBEAT_H
