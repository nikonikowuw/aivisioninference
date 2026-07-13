#ifndef AIVISION_MONITOR_HEARTBEAT_REPORTER_H
#define AIVISION_MONITOR_HEARTBEAT_REPORTER_H

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <memory>
#include <mutex>
#include <string>
#include <thread>

namespace aivision
{
    class InferenceEngine;

    namespace monitor
    {
        class DeviceMonitor;
        class MetricsFlattener;

        class HeartbeatReporter
        {
        public:
            explicit HeartbeatReporter(InferenceEngine* engine);
            ~HeartbeatReporter();

            /// 启动心跳上报线程
            void Start();

            /// 停止心跳上报线程
            void Stop();

            /// 是否运行中
            bool IsRunning() const { return running_.load(); }

            /// 关联 DeviceMonitor 以获取完整设备快照
            void SetDeviceMonitor(DeviceMonitor* monitor) { device_monitor_ = monitor; }

        private:
            /// 心跳上报循环
            void ReportLoop();

            /// 发送心跳请求并解析响应
            void SendHeartbeat();

            /// 构造心跳 JSON 字符串
            std::string BuildHeartbeatPayload();

            /// 解析心跳响应并触发算法部署
            void ParseAndDeploy(const std::string& response_json);

            InferenceEngine* engine_;
            DeviceMonitor* device_monitor_ = nullptr;
            std::unique_ptr<MetricsFlattener> metrics_flattener_;
            uint64_t last_flatten_timestamp_ms_ = 0;
            std::atomic<bool> running_{false};
            std::unique_ptr<std::thread> thread_;
            std::mutex stop_mutex_;
            std::condition_variable stop_cv_;
            std::chrono::steady_clock::time_point start_time_;
        };
    }
}

#endif // AIVISION_MONITOR_HEARTBEAT_REPORTER_H
