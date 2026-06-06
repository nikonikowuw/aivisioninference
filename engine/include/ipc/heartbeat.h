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

        class IPCServer;

        /// 心跳管理器
        class HeartbeatManager
        {
        public:
            using TimeoutCallback = std::function<void()>;

            HeartbeatManager();

            /// 构造并绑定 IPCServer
            explicit HeartbeatManager(IPCServer *server);

            /// 启动心跳监控线程
            void Start();

            /// 停止心跳监控
            void Stop();

            /// 刷新心跳计时器 (收到 Go 心跳时调用)
            void Refresh();

            /// 设置心跳超时回调
            void SetTimeoutCallback(TimeoutCallback cb);

            /// 获取引擎运行时长 (秒)
            uint64_t GetUptimeSec() const;

            /// 获取心跳序列号
            uint64_t GetSequence() const { return sequence_.load(); }

        private:
            /// 心跳监控循环
            void MonitorLoop();

            IPCServer *server_{nullptr};
            std::atomic<bool> running_{false};
            std::atomic<uint64_t> sequence_{0};
            std::chrono::steady_clock::time_point start_time_;
            std::chrono::steady_clock::time_point last_heartbeat_;
            int timeout_sec_{30};
            std::unique_ptr<std::thread> monitor_thread_;
            TimeoutCallback timeout_cb_;

        public:
            /// 绑定 IPCServer (延迟绑定)
            void SetServer(IPCServer *server) { server_ = server; }
        };

    } // namespace ipc
} // namespace aivision

#endif // AIVISION_IPC_HEARTBEAT_H
