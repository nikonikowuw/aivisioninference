#include "monitor/command_executor.h"
#include "logger/logger.h"

#include <chrono>
#include <cstdio>
#include <cstring>
#include <memory>
#include <sstream>
#include <thread>
#include <vector>

#include <signal.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#include <fcntl.h>

namespace aivision
{
    namespace monitor
    {
        CommandExecutor::CommandExecutor()
        {
        }

        CommandExecutor::~CommandExecutor()
        {
        }

        void CommandExecutor::SetResultCallback(ResultCallback callback)
        {
            result_callback_ = callback;
        }

        ShellExecResult CommandExecutor::Execute(const ShellExecRequest& request)
        {
            auto start_time = std::chrono::steady_clock::now();

            // Use fork+exec for better process control
            ShellExecResult result = ExecuteForkExec(request);

            auto end_time = std::chrono::steady_clock::now();
            result.duration_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                end_time - start_time).count();

            // Invoke callback if set
            if (result_callback_)
            {
                result_callback_(result, request);
            }

            return result;
        }

        ShellExecResult CommandExecutor::ExecuteForkExec(const ShellExecRequest& request)
        {
            ShellExecResult result;

            // Create pipes for stdout and stderr
            int stdout_pipe[2];
            int stderr_pipe[2];

            if (pipe(stdout_pipe) < 0 || pipe(stderr_pipe) < 0)
            {
                result.error_message = "Failed to create pipes";
                return result;
            }

            pid_t pid = fork();

            if (pid < 0)
            {
                // Fork failed
                close(stdout_pipe[0]);
                close(stdout_pipe[1]);
                close(stderr_pipe[0]);
                close(stderr_pipe[1]);
                result.error_message = "Failed to fork process";
                return result;
            }

            if (pid == 0)
            {
                // Child process
                close(stdout_pipe[0]); // Close read end
                close(stderr_pipe[0]);

                // Redirect stdout and stderr
                dup2(stdout_pipe[1], STDOUT_FILENO);
                dup2(stderr_pipe[1], STDERR_FILENO);

                // Close original pipe write ends after dup
                close(stdout_pipe[1]);
                close(stderr_pipe[1]);

                // Execute command via /bin/sh -c
                execl("/bin/sh", "sh", "-c", request.command.c_str(), nullptr);

                // If execl returns, it failed
                // Note: async-signal-safe write(2) — LOG_ERROR is unsafe after fork
                static const char fail_msg[] = "[CommandExecutor] execl failed: ";
                (void)::write(STDERR_FILENO, fail_msg, sizeof(fail_msg) - 1);
                (void)::write(STDERR_FILENO, ::strerror(errno), ::strlen(::strerror(errno)));
                (void)::write(STDERR_FILENO, "\n", 1);
                _exit(127);
            }

            // Parent process
            close(stdout_pipe[1]); // Close write ends
            close(stderr_pipe[1]);

            // Set stdout pipe to non-blocking for timeout handling
            int stdout_flags = fcntl(stdout_pipe[0], F_GETFL, 0);
            fcntl(stdout_pipe[0], F_SETFL, stdout_flags | O_NONBLOCK);

            int stderr_flags = fcntl(stderr_pipe[0], F_GETFL, 0);
            fcntl(stderr_pipe[0], F_SETFL, stderr_flags | O_NONBLOCK);

            // Read output in a separate thread to avoid deadlocks
            std::string stdout_data, stderr_data;
            std::thread reader([&]() {
                char buffer[4096];
                ssize_t bytes;
                while (true)
                {
                    bytes = read(stdout_pipe[0], buffer, sizeof(buffer) - 1);
                    if (bytes > 0)
                    {
                        buffer[bytes] = '\0';
                        stdout_data += buffer;
                    }
                    else if (bytes < 0 && errno == EAGAIN)
                    {
                        // Non-blocking pipe has no data yet — sleep briefly
                        // and retry so we don't exit the loop prematurely.
                        std::this_thread::sleep_for(std::chrono::milliseconds(10));
                        continue;
                    }
                    else
                    {
                        // EOF (child closed pipe) or unrecoverable error
                        break;
                    }
                }
            });

            std::thread stderr_reader([&]() {
                char buffer[4096];
                ssize_t bytes;
                while (true)
                {
                    bytes = read(stderr_pipe[0], buffer, sizeof(buffer) - 1);
                    if (bytes > 0)
                    {
                        buffer[bytes] = '\0';
                        stderr_data += buffer;
                    }
                    else if (bytes < 0 && errno == EAGAIN)
                    {
                        std::this_thread::sleep_for(std::chrono::milliseconds(10));
                        continue;
                    }
                    else
                    {
                        break;
                    }
                }
            });

            // Wait for child with timeout
            int timeout_ms = request.timeout_seconds * 1000;
            int elapsed_ms = 0;
            const int CHECK_INTERVAL_MS = 100;
            int status = 0;

            while (elapsed_ms < timeout_ms)
            {
                pid_t ret = waitpid(pid, &status, WNOHANG);
                if (ret == pid)
                {
                    // Process completed
                    break;
                }
                std::this_thread::sleep_for(std::chrono::milliseconds(CHECK_INTERVAL_MS));
                elapsed_ms += CHECK_INTERVAL_MS;
            }

            if (elapsed_ms >= timeout_ms)
            {
                // Timeout: kill child process
                kill(pid, SIGKILL);
                waitpid(pid, &status, 0);
                result.success = false;
                result.exit_code = -1;
                result.error_message = "Command timed out after " + std::to_string(request.timeout_seconds) + "s";
            }
            else if (WIFEXITED(status))
            {
                result.exit_code = WEXITSTATUS(status);
                result.success = (result.exit_code == 0);
            }
            else if (WIFSIGNALED(status))
            {
                result.exit_code = -128 + WTERMSIG(status);
                result.success = false;
                result.error_message = "Process terminated by signal " + std::to_string(WTERMSIG(status));
            }

            // Close pipes to signal readers to finish
            close(stdout_pipe[0]);
            close(stderr_pipe[0]);

            // Wait for reader threads
            if (reader.joinable())
                reader.join();
            if (stderr_reader.joinable())
                stderr_reader.join();

            result.stdout_str = stdout_data;
            result.stderr_str = stderr_data;

            return result;
        }

    } // namespace monitor
} // namespace aivision
