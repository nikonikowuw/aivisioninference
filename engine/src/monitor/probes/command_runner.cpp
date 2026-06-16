#include "probes/device_probe.h"

#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/select.h>
#include <signal.h>
#include <fcntl.h>
#include <cstring>
#include <chrono>
#include <algorithm>
#include <iostream>

namespace aivision
{
    namespace monitor
    {
        CommandResult CommandRunner::Run(const std::vector<std::string>& args, uint32_t timeout_ms, size_t max_output_bytes)
        {
            CommandResult result;
            if (args.empty())
            {
                result.error = "Empty command args";
                return result;
            }

            int out_pipe[2];
            int err_pipe[2];
            if (pipe(out_pipe) < 0)
            {
                result.error = "Failed to create stdout pipe: " + std::string(strerror(errno));
                return result;
            }
            if (pipe(err_pipe) < 0)
            {
                close(out_pipe[0]);
                close(out_pipe[1]);
                result.error = "Failed to create stderr pipe: " + std::string(strerror(errno));
                return result;
            }

            pid_t pid = fork();
            if (pid < 0)
            {
                close(out_pipe[0]);
                close(out_pipe[1]);
                close(err_pipe[0]);
                close(err_pipe[1]);
                result.error = "Failed to fork: " + std::string(strerror(errno));
                return result;
            }

            if (pid == 0)
            {
                // Child process
                dup2(out_pipe[1], STDOUT_FILENO);
                dup2(err_pipe[1], STDERR_FILENO);

                close(out_pipe[0]);
                close(out_pipe[1]);
                close(err_pipe[0]);
                close(err_pipe[1]);

                std::vector<char*> argv;
                for (const auto& arg : args)
                {
                    argv.push_back(const_cast<char*>(arg.c_str()));
                }
                argv.push_back(nullptr);

                // Ensure child doesn't inherit signals or unnecessary FDs if needed
                // But for monitor probes, execvp is standard.
                execvp(argv[0], argv.data());
                std::cerr << "execvp failed: " << strerror(errno) << std::endl;
                _exit(127);
            }

            // Parent process
            close(out_pipe[1]);
            close(err_pipe[1]);

            fcntl(out_pipe[0], F_SETFL, O_NONBLOCK);
            fcntl(err_pipe[0], F_SETFL, O_NONBLOCK);

            int out_fd = out_pipe[0];
            int err_fd = err_pipe[0];

            bool timed_out = false;
            auto start_time = std::chrono::steady_clock::now();
            uint64_t elapsed_ms = 0;

            while (elapsed_ms < timeout_ms)
            {
                fd_set read_fds;
                FD_ZERO(&read_fds);
                int max_fd = -1;
                if (out_fd >= 0)
                {
                    FD_SET(out_fd, &read_fds);
                    if (out_fd > max_fd) max_fd = out_fd;
                }
                if (err_fd >= 0)
                {
                    FD_SET(err_fd, &read_fds);
                    if (err_fd > max_fd) max_fd = err_fd;
                }

                if (max_fd == -1)
                {
                    break;
                }

                struct timeval tv;
                uint64_t remaining_ms = timeout_ms - elapsed_ms;
                tv.tv_sec = remaining_ms / 1000;
                tv.tv_usec = (remaining_ms % 1000) * 1000;

                int select_ret = select(max_fd + 1, &read_fds, nullptr, nullptr, &tv);
                if (select_ret < 0)
                {
                    if (errno == EINTR)
                    {
                        elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                            std::chrono::steady_clock::now() - start_time).count();
                        continue;
                    }
                    result.error = "Select error: " + std::string(strerror(errno));
                    break;
                }

                if (select_ret == 0)
                {
                    timed_out = true;
                    break;
                }

                char buffer[4096];
                if (out_fd >= 0 && FD_ISSET(out_fd, &read_fds))
                {
                    ssize_t bytes_read = read(out_fd, buffer, sizeof(buffer));
                    if (bytes_read > 0)
                    {
                        if (result.stdout_data.size() < max_output_bytes)
                        {
                            size_t to_append = std::min(static_cast<size_t>(bytes_read), max_output_bytes - result.stdout_data.size());
                            result.stdout_data.append(buffer, to_append);
                        }
                    }
                    else if (bytes_read == 0 || (bytes_read < 0 && errno != EAGAIN && errno != EWOULDBLOCK))
                    {
                        close(out_fd);
                        out_fd = -1;
                    }
                }

                if (err_fd >= 0 && FD_ISSET(err_fd, &read_fds))
                {
                    ssize_t bytes_read = read(err_fd, buffer, sizeof(buffer));
                    if (bytes_read > 0)
                    {
                        if (result.stderr_data.size() < max_output_bytes)
                        {
                            size_t to_append = std::min(static_cast<size_t>(bytes_read), max_output_bytes - result.stderr_data.size());
                            result.stderr_data.append(buffer, to_append);
                        }
                    }
                    else if (bytes_read == 0 || (bytes_read < 0 && errno != EAGAIN && errno != EWOULDBLOCK))
                    {
                        close(err_fd);
                        err_fd = -1;
                    }
                }

                elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::steady_clock::now() - start_time).count();
            }

            if (out_fd >= 0) close(out_fd);
            if (err_fd >= 0) close(err_fd);

            if (timed_out)
            {
                result.error = "Command timed out after " + std::to_string(timeout_ms) + "ms";
                kill(pid, SIGKILL);
                int status = 0;
                waitpid(pid, &status, 0);
            }
            else
            {
                int status = 0;
                pid_t wait_ret = waitpid(pid, &status, WNOHANG);
                if (wait_ret == 0)
                {
                    kill(pid, SIGKILL);
                    waitpid(pid, &status, 0);
                    result.error = "Process did not terminate after reading all pipes";
                }
                else if (wait_ret > 0)
                {
                    if (WIFEXITED(status))
                    {
                        result.exit_code = WEXITSTATUS(status);
                        result.success = (result.exit_code == 0);
                    }
                    else if (WIFSIGNALED(status))
                    {
                        result.error = "Process killed by signal " + std::to_string(WTERMSIG(status));
                    }
                }
                else
                {
                    result.error = "waitpid error: " + std::string(strerror(errno));
                }
            }

            return result;
        }
    }
}
