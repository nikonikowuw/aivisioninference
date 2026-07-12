#include "monitor/pty_module.h"

#include <cstring>
#include <iostream>
#include <sstream>

#include <fcntl.h>
#include <signal.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>

namespace aivision
{
    namespace monitor
    {
        PTYModule::PTYModule()
        {
        }

        PTYModule::~PTYModule()
        {
            running_.store(false);
            // Close all sessions
            std::lock_guard<std::mutex> lock(mutex_);
            auto sessions_copy = sessions_;
            for (const auto& [session_id, session] : sessions_copy)
            {
                CleanupSession(session_id);
            }
            sessions_.clear();
        }

        void PTYModule::SetOutputCallback(PtyOutputCallback callback)
        {
            output_callback_ = callback;
        }

        void PTYModule::SetCloseCallback(PtyCloseCallback callback)
        {
            close_callback_ = callback;
        }

        size_t PTYModule::ActiveSessionCount() const
        {
            std::lock_guard<std::mutex> lock(mutex_);
            size_t count = 0;
            for (const auto& [_, session] : sessions_)
            {
                if (session->active)
                    count++;
            }
            return count;
        }

        bool PTYModule::OpenSession(const std::string& session_id)
        {
            // Open PTY master
            int master_fd = open("/dev/ptmx", O_RDWR | O_NOCTTY);
            if (master_fd < 0)
            {
                std::cerr << "[PTYModule] Failed to open /dev/ptmx: " << strerror(errno) << std::endl;
                return false;
            }

            // Grant access and unlock slave
            if (grantpt(master_fd) < 0 || unlockpt(master_fd) < 0)
            {
                std::cerr << "[PTYModule] grantpt/unlockpt failed: " << strerror(errno) << std::endl;
                close(master_fd);
                return false;
            }

            // Get slave name
            const char* slave_name = ptsname(master_fd);
            if (!slave_name)
            {
                std::cerr << "[PTYModule] ptsname failed: " << strerror(errno) << std::endl;
                close(master_fd);
                return false;
            }

            // Fork child process
            pid_t pid = fork();
            if (pid < 0)
            {
                std::cerr << "[PTYModule] fork failed: " << strerror(errno) << std::endl;
                close(master_fd);
                return false;
            }

            if (pid == 0)
            {
                // Child process: create new session and start shell
                close(master_fd);

                // Create new session
                if (setsid() < 0)
                {
                    _exit(1);
                }

                // Open slave as controlling terminal
                int slave_fd = open(slave_name, O_RDWR);
                if (slave_fd < 0)
                {
                    _exit(1);
                }

                // Set up stdin/stdout/stderr to slave
                dup2(slave_fd, STDIN_FILENO);
                dup2(slave_fd, STDOUT_FILENO);
                dup2(slave_fd, STDERR_FILENO);

                if (slave_fd > STDERR_FILENO)
                {
                    close(slave_fd);
                }

                // Set TERM environment for proper terminal support
                setenv("TERM", "xterm-256color", 1);

                // Start shell
                execl("/bin/sh", "sh", "-l", nullptr);
                // If execl fails
                _exit(127);
            }

            // Parent process
            // Set up the session
            auto session = std::make_shared<PTYSession>();
            session->master_fd = master_fd;
            session->child_pid = pid;
            session->session_id = session_id;
            session->active = true;

            {
                std::lock_guard<std::mutex> lock(mutex_);
                sessions_[session_id] = session;
            }

            // Start reader thread for this session and detach it.
            // The thread reads PTY output and invokes output_callback_. It exits
            // naturally when the master fd is closed (during session cleanup).
            std::thread reader(&PTYModule::ReadLoop, this, session_id, master_fd);
            reader.detach();

            std::cout << "[PTYModule] Session opened: " << session_id
                      << " (slave=" << slave_name << ", pid=" << pid << ")" << std::endl;

            return true;
        }

        bool PTYModule::WriteToSession(const std::string& session_id, const std::string& data)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = sessions_.find(session_id);
            if (it == sessions_.end() || !it->second->active)
            {
                std::cerr << "[PTYModule] Write to inactive session: " << session_id << std::endl;
                return false;
            }

            int fd = it->second->master_fd;
            const char* buf = data.data();
            size_t len = data.size();
            size_t total_written = 0;

            while (total_written < len)
            {
                ssize_t written = write(fd, buf + total_written, len - total_written);
                if (written < 0)
                {
                    if (errno == EINTR) continue;
                    std::cerr << "[PTYModule] Write failed: " << strerror(errno) << std::endl;
                    return false;
                }
                total_written += written;
            }

            return true;
        }

        bool PTYModule::ResizeSession(const std::string& session_id, unsigned short cols, unsigned short rows)
        {
            std::lock_guard<std::mutex> lock(mutex_);

            auto it = sessions_.find(session_id);
            if (it == sessions_.end() || !it->second->active)
            {
                return false;
            }

            struct winsize ws;
            memset(&ws, 0, sizeof(ws));
            ws.ws_col = cols;
            ws.ws_row = rows;
            ws.ws_xpixel = 0;
            ws.ws_ypixel = 0;

            if (ioctl(it->second->master_fd, TIOCSWINSZ, &ws) < 0)
            {
                std::cerr << "[PTYModule] Resize failed: " << strerror(errno) << std::endl;
                return false;
            }

            return true;
        }

        void PTYModule::CloseSession(const std::string& session_id)
        {
            {
                std::lock_guard<std::mutex> lock(mutex_);
                CleanupSession(session_id);
            }

            if (close_callback_)
            {
                close_callback_(session_id, "session closed");
            }
        }

        void PTYModule::ReadLoop(const std::string& session_id, int fd)
        {
            char buffer[4096];
            while (running_.load())
            {
                fd_set read_fds;
                FD_ZERO(&read_fds);
                FD_SET(fd, &read_fds);

                struct timeval timeout;
                timeout.tv_sec = 0;
                timeout.tv_usec = 100000; // 100ms

                int ret = select(fd + 1, &read_fds, nullptr, nullptr, &timeout);
                if (ret < 0)
                {
                    if (errno == EINTR) continue;
                    break;
                }

                if (ret == 0) continue; // Timeout, loop again

                if (FD_ISSET(fd, &read_fds))
                {
                    ssize_t bytes_read = read(fd, buffer, sizeof(buffer) - 1);
                    if (bytes_read > 0)
                    {
                        buffer[bytes_read] = '\0';
                        std::string data(buffer, bytes_read);

                        if (output_callback_)
                        {
                            output_callback_(session_id, data);
                        }
                    }
                    else
                    {
                        // EOF or error — child exited or master_fd closed externally.
                        // Do NOT call CloseSession/CleanupSession here: this thread
                        // IS the reader thread, and CleanupSession is called by the
                        // external MQTT pty_close handler or the destructor, which
                        // closes the master fd (causing this loop to break).
                        break;
                    }
                }
            }

            // Notify that the reader has exited (fire-and-forget from this thread).
            // The session struct will be cleaned up by the external close path.
            if (close_callback_)
            {
                close_callback_(session_id, "pty reader exited");
            }
        }

        void PTYModule::CleanupSession(const std::string& session_id)
        {
            auto it = sessions_.find(session_id);
            if (it == sessions_.end())
                return;

            auto session = it->second;
            session->active = false;

            // Close master fd — the detached reader thread's select() will
            // return and the thread will exit naturally. No join is needed.
            if (session->master_fd >= 0)
            {
                close(session->master_fd);
                session->master_fd = -1;
            }

            // Kill child process
            if (session->child_pid > 0)
            {
                kill(session->child_pid, SIGKILL);
                // Reap child (WNOHANG: child may already be reaped)
                int status;
                waitpid(session->child_pid, &status, WNOHANG);
                session->child_pid = -1;
            }

            sessions_.erase(session_id);

            std::cout << "[PTYModule] Session cleaned up: " << session_id << std::endl;
        }

    } // namespace monitor
} // namespace aivision
