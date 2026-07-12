#ifndef AIVISION_MONITOR_PTY_MODULE_H
#define AIVISION_MONITOR_PTY_MODULE_H

#include <atomic>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <thread>

namespace aivision
{
    namespace monitor
    {
        /// PTY 会话信息
        struct PTYSession {
            int master_fd = -1;        // PTY master fd
            pid_t child_pid = -1;       // 子进程 PID
            std::string session_id;     // 会话唯一标识
            bool active = false;        // 是否活跃
        };

        /// PTY 输出回调：接收来自 PTY 的数据，需要由上层转发到 MQTT
        using PtyOutputCallback = std::function<void(const std::string& session_id, const std::string& data)>;

        /// PTY 错误/关闭回调
        using PtyCloseCallback = std::function<void(const std::string& session_id, const std::string& error)>;

        /// PTYModule 管理远程 Web 终端的 PTY 会话
        /// 每个会话对应一个独立的 /dev/ptmx PTY 实例
        class PTYModule
        {
        public:
            PTYModule();
            ~PTYModule();

            /// 创建一个新的 PTY 会话（对应 pty_open 命令）
            /// @param session_id 会话 ID
            /// @return true 如果创建成功
            bool OpenSession(const std::string& session_id);

            /// 向 PTY 写入数据（对应 pty_write 命令）
            /// @param session_id 会话 ID
            /// @param data 要写入的数据
            /// @return true 如果写入成功
            bool WriteToSession(const std::string& session_id, const std::string& data);

            /// 调整 PTY 窗口大小（对应 pty_resize 命令）
            /// @param session_id 会话 ID
            /// @param cols 列数
            /// @param rows 行数
            /// @return true 如果调整成功
            bool ResizeSession(const std::string& session_id, unsigned short cols, unsigned short rows);

            /// 关闭一个 PTY 会话（对应 pty_close 命令）
            /// @param session_id 会话 ID
            void CloseSession(const std::string& session_id);

            /// 设置输出回调
            void SetOutputCallback(PtyOutputCallback callback);

            /// 设置关闭回调
            void SetCloseCallback(PtyCloseCallback callback);

            /// 获取活跃会话数
            size_t ActiveSessionCount() const;

        private:
            /// 内部读线程：从 PTY master fd 读取输出
            void ReadLoop(const std::string& session_id, int fd);

            /// 清理会话资源
            void CleanupSession(const std::string& session_id);

            mutable std::mutex mutex_;
            std::map<std::string, std::shared_ptr<PTYSession>> sessions_;
            std::map<std::string, std::shared_ptr<std::thread>> reader_threads_;
            std::atomic<bool> running_{true};

            PtyOutputCallback output_callback_;
            PtyCloseCallback close_callback_;
        };

    } // namespace monitor
} // namespace aivision

#endif // AIVISION_MONITOR_PTY_MODULE_H
