#ifndef AIVISION_MONITOR_COMMAND_EXECUTOR_H
#define AIVISION_MONITOR_COMMAND_EXECUTOR_H

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <string>
#include <thread>
#include <vector>

namespace aivision
{
    namespace monitor
    {
        /// Shell 命令执行结果
        struct ShellExecResult {
            bool success = false;
            std::string stdout_str;
            std::string stderr_str;
            int exit_code = -1;
            int64_t duration_ms = 0;
            std::string error_message;
        };

        /// Shell 执行请求参数
        struct ShellExecRequest {
            std::string trace_id;
            std::string execution_id;
            std::string command;
            int timeout_seconds = 30;
            std::string callback_url;
        };

        /// CommandExecutor 负责接收 shell_exec 命令并在独立进程中执行
        class CommandExecutor
        {
        public:
            CommandExecutor();
            ~CommandExecutor();

            /// 在子进程中执行 shell 命令并返回结果
            /// @param request 执行请求参数
            /// @return 执行结果（stdout、stderr、exit_code、duration）
            ShellExecResult Execute(const ShellExecRequest& request);

            /// 设置结果回调函数（引擎层注入，负责 HTTP POST 回 Go 控制面）
            using ResultCallback = std::function<void(const ShellExecResult&, const ShellExecRequest&)>;
            void SetResultCallback(ResultCallback callback);

        private:
            /// 使用 popen + fork 执行命令，带超时控制
            ShellExecResult ExecuteWithTimeout(const ShellExecRequest& request);

            /// 使用 fork+exec 方式执行（更精确的进程控制）
            ShellExecResult ExecuteForkExec(const ShellExecRequest& request);

            ResultCallback result_callback_;
        };

    } // namespace monitor
} // namespace aivision

#endif // AIVISION_MONITOR_COMMAND_EXECUTOR_H
