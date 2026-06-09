// IPC Server 实现
// TODO: flatc 生成 C++ 代码后填充 FlatBuffers 序列化/反序列化

#include "ipc/ipc_server.h"
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <algorithm>
#include <cstring>
#include <iostream>
#include <mutex>

namespace aivision
{
    namespace ipc
    {

        static bool ParseAddr(const std::string &addr, std::string &host, int &port)
        {
            auto pos = addr.rfind(':');
            if (pos == std::string::npos || pos == 0)
                return false;
            host = addr.substr(0, pos);
            try { port = std::stoi(addr.substr(pos + 1)); }
            catch (...) { return false; }
            return port > 0 && port <= 65535;
        }

        IPCServer::IPCServer(const IPCServerConfig &config)
            : config_(config) {}

        IPCServer::~IPCServer() { Stop(); }

        bool IPCServer::Start()
        {
            // 解析 host:port
            std::string host;
            int port;
            if (!ParseAddr(config_.addr, host, port))
            {
                std::cerr << "Invalid addr: " << config_.addr << std::endl;
                return false;
            }

            // 创建 TCP socket
            server_fd_ = socket(AF_INET, SOCK_STREAM, 0);
            if (server_fd_ < 0)
            {
                std::cerr << "Failed to create TCP socket" << std::endl;
                return false;
            }

            // 允许端口复用
            int opt = 1;
            setsockopt(server_fd_, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

            // 绑定地址
            struct sockaddr_in sin;
            memset(&sin, 0, sizeof(sin));
            sin.sin_family = AF_INET;
            sin.sin_port = htons(static_cast<uint16_t>(port));
            inet_pton(AF_INET, host.c_str(), &sin.sin_addr);

            if (bind(server_fd_, (struct sockaddr *)&sin, sizeof(sin)) < 0)
            {
                std::cerr << "Failed to bind TCP socket: "
                          << config_.addr << std::endl;
                close(server_fd_);
                server_fd_ = -1;
                return false;
            }

            // 监听
            if (listen(server_fd_, config_.backlog) < 0)
            {
                std::cerr << "Failed to listen on TCP socket" << std::endl;
                close(server_fd_);
                server_fd_ = -1;
                return false;
            }

            running_.store(true);

            // 启动接受连接线程
            accept_thread_ = std::make_unique<std::thread>(&IPCServer::AcceptLoop, this);

            return true;
        }

        void IPCServer::Stop()
        {
            running_.store(false);

            // 关闭 server_fd 使 accept() 返回错误退出循环
            if (server_fd_ >= 0)
            {
                shutdown(server_fd_, SHUT_RDWR);
                close(server_fd_);
                server_fd_ = -1;
            }

            // 关闭所有已连接的客户端 fd，使阻塞的 read() 返回
            {
                std::lock_guard<std::mutex> lock(clients_mutex_);
                for (int fd : client_fds_)
                {
                    shutdown(fd, SHUT_RDWR);
                    close(fd);
                }
                client_fds_.clear();
            }

            // 等待 accept 线程退出
            if (accept_thread_ && accept_thread_->joinable())
            {
                accept_thread_->join();
            }

            // 等待所有客户端处理线程退出
            {
                std::vector<std::unique_ptr<std::thread>> threads;
                {
                    std::lock_guard<std::mutex> lock(threads_mutex_);
                    threads.swap(client_threads_);
                }
                for (auto &t : threads)
                {
                    if (t && t->joinable())
                    {
                        t->join();
                    }
                }
            }
        }

        static thread_local int t_active_client_fd = -1;

        int IPCServer::GetActiveClientFd()
        {
            return t_active_client_fd;
        }

        bool IPCServer::SendResponse(int client_fd, uint32_t resp_type, const uint8_t *payload, size_t payload_len)
        {
            if (client_fd < 0) return false;
            uint8_t header[8];
            std::memcpy(header, &resp_type, sizeof(uint32_t));
            std::memcpy(header + 4, &payload_len, sizeof(uint32_t));

            if (write(client_fd, header, 8) != 8)
            {
                return false;
            }
            if (payload_len > 0 && payload)
            {
                if (write(client_fd, payload, payload_len) != static_cast<ssize_t>(payload_len))
                {
                    return false;
                }
            }
            return true;
        }

        void IPCServer::RegisterHandler(uint16_t signal_type, CommandHandler handler)
        {
            std::lock_guard<std::mutex> lock(handlers_mutex_);
            handlers_[signal_type] = std::move(handler);
        }

        bool IPCServer::SendMessage(uint16_t signal_type,
                                    flatbuffers::FlatBufferBuilder &fbb)
        {
            (void)signal_type; (void)fbb;
            // TODO: 包装 IPCEnvelope 并发送到 Go 侧
            return true;
        }

        bool IPCServer::BroadcastMessage(uint16_t signal_type,
                                         flatbuffers::FlatBufferBuilder &fbb)
        {
            (void)signal_type; (void)fbb;
            // TODO: 广播到所有已连接的客户端
            return true;
        }

        void IPCServer::AcceptLoop()
        {
            while (running_.load())
            {
                struct sockaddr_in client_addr;
                socklen_t client_len = sizeof(client_addr);
                int client_fd = accept(server_fd_,
                                       (struct sockaddr *)&client_addr,
                                       &client_len);
                if (client_fd < 0)
                {
                    if (running_.load())
                    {
                        std::cerr << "Accept failed" << std::endl;
                    }
                    break;
                }

                {
                    std::lock_guard<std::mutex> lock(clients_mutex_);
                    client_fds_.push_back(client_fd);
                }

                // 为每个客户端创建处理线程 (存储线程对象，Stop 时 join)
                {
                    std::lock_guard<std::mutex> lock(threads_mutex_);
                    client_threads_.emplace_back(
                        std::make_unique<std::thread>(&IPCServer::HandleClient, this, client_fd));
                }
            }
        }

        void IPCServer::HandleClient(int client_fd)
        {
            t_active_client_fd = client_fd;
            auto buffer = std::make_unique<uint8_t[]>(config_.recv_buffer_size);

            while (running_.load())
            {
                ssize_t n = read(client_fd, buffer.get(), config_.recv_buffer_size);
                if (n <= 0)
                {
                    break;
                }
                ProcessMessage(client_fd, buffer.get(), static_cast<size_t>(n));
            }

            close(client_fd);
            t_active_client_fd = -1;

            // 从客户端列表移除
            std::lock_guard<std::mutex> lock(clients_mutex_);
            auto it = std::find(client_fds_.begin(), client_fds_.end(), client_fd);
            if (it != client_fds_.end())
            {
                client_fds_.erase(it);
            }
        }

        bool IPCServer::ProcessMessage(int /*client_fd*/, uint8_t *buffer, size_t size)
        {
            // 信封格式：[4字节命令类型][4字节payload长度][payload]
            constexpr size_t kMinMessageSize = 8;

            if (!buffer || size < kMinMessageSize)
            {
                std::cerr << "[IPC] Invalid message: size=" << size
                          << ", min=" << kMinMessageSize << std::endl;
                return false;
            }

            uint32_t cmd_type = 0;
            std::memcpy(&cmd_type, buffer, sizeof(uint32_t));

            uint32_t payload_len = 0;
            std::memcpy(&payload_len, buffer + 4, sizeof(uint32_t));

            if (size < kMinMessageSize + payload_len)
            {
                std::cerr << "[IPC] Incomplete message: size=" << size
                          << ", expected=" << kMinMessageSize + payload_len << std::endl;
                return false;
            }

            const uint8_t *payload = buffer + kMinMessageSize;
            size_t payload_size = payload_len;

            CommandHandler handler;
            {
                std::lock_guard<std::mutex> lock(handlers_mutex_);
                auto it = handlers_.find(cmd_type);
                if (it != handlers_.end())
                {
                    handler = it->second;
                }
            }

            if (!handler)
            {
                std::cerr << "[IPC] No handler for cmd_type=" << cmd_type << std::endl;
                return true;
            }

            try
            {
                handler(payload, payload_size, 0);
            }
            catch (const std::exception &e)
            {
                std::cerr << "[IPC] Handler exception for cmd_type="
                          << cmd_type << ": " << e.what() << std::endl;
                return false;
            }

            return true;
        }

    } // namespace ipc
} // namespace aivision
