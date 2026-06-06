// IPC Server 实现
// TODO: flatc 生成 C++ 代码后填充 FlatBuffers 序列化/反序列化

#include "ipc/ipc_server.h"
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <cstring>
#include <iostream>
#include <algorithm>

namespace aivision
{
    namespace ipc
    {

        IPCServer::IPCServer(const IPCServerConfig &config)
            : config_(config) {}

        IPCServer::~IPCServer() { Stop(); }

        bool IPCServer::Start()
        {
            // 创建 UDS socket
            server_fd_ = socket(AF_UNIX, SOCK_STREAM, 0);
            if (server_fd_ < 0)
            {
                std::cerr << "Failed to create UDS socket" << std::endl;
                return false;
            }

            // 移除已存在的 socket 文件
            unlink(config_.socket_path.c_str());

            // 绑定地址
            struct sockaddr_un addr;
            memset(&addr, 0, sizeof(addr));
            addr.sun_family = AF_UNIX;
            strncpy(addr.sun_path, config_.socket_path.c_str(),
                    sizeof(addr.sun_path) - 1);

            if (bind(server_fd_, (struct sockaddr *)&addr, sizeof(addr)) < 0)
            {
                std::cerr << "Failed to bind UDS socket: "
                          << config_.socket_path << std::endl;
                close(server_fd_);
                server_fd_ = -1;
                return false;
            }

            // 监听
            if (listen(server_fd_, config_.backlog) < 0)
            {
                std::cerr << "Failed to listen on UDS socket" << std::endl;
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
                close(server_fd_);
                server_fd_ = -1;
            }

            // 关闭所有已连接的客户端 fd，使阻塞的 read() 返回
            {
                std::lock_guard<std::mutex> lock(clients_mutex_);
                for (int fd : client_fds_)
                {
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
                std::lock_guard<std::mutex> lock(threads_mutex_);
                for (auto &t : client_threads_)
                {
                    if (t && t->joinable())
                    {
                        t->join();
                    }
                }
                client_threads_.clear();
            }

            // 清理 socket 文件
            unlink(config_.socket_path.c_str());
        }

        void IPCServer::RegisterHandler(uint16_t signal_type, CommandHandler handler)
        {
            std::lock_guard<std::mutex> lock(handlers_mutex_);
            handlers_[signal_type] = std::move(handler);
        }

        bool IPCServer::SendMessage(uint16_t signal_type,
                                    flatbuffers::FlatBufferBuilder &fbb)
        {
            // TODO: 包装 IPCEnvelope 并发送到 Go 侧
            return true;
        }

        bool IPCServer::BroadcastMessage(uint16_t signal_type,
                                         flatbuffers::FlatBufferBuilder &fbb)
        {
            // TODO: 广播到所有已连接的客户端
            return true;
        }

        void IPCServer::AcceptLoop()
        {
            while (running_.load())
            {
                struct sockaddr_un client_addr;
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

            // 从客户端列表移除
            std::lock_guard<std::mutex> lock(clients_mutex_);
            auto it = std::find(client_fds_.begin(), client_fds_.end(), client_fd);
            if (it != client_fds_.end())
            {
                client_fds_.erase(it);
            }
        }

        bool IPCServer::ProcessMessage(int client_fd, uint8_t *buffer, size_t size)
        {
            // 最小消息大小: 4 字节头 (2字节 signal_type + 2字节 reserved)
            // + 8 字节 sequence_id = 12 字节
            constexpr size_t kMinMessageSize = 12;

            if (!buffer || size < kMinMessageSize)
            {
                std::cerr << "[IPC] Invalid message: size=" << size
                          << ", min=" << kMinMessageSize << std::endl;
                return false;
            }

            // 解析消息头
            uint16_t signal_type = 0;
            std::memcpy(&signal_type, buffer, sizeof(uint16_t));

            uint64_t sequence_id = 0;
            std::memcpy(&sequence_id, buffer + 4, sizeof(uint64_t));

            // payload 指向消息头之后的数据
            const uint8_t *payload = buffer + kMinMessageSize;
            size_t payload_size = size - kMinMessageSize;

            // 查找并分发到注册的 Handler
            std::lock_guard<std::mutex> lock(handlers_mutex_);
            auto it = handlers_.find(signal_type);
            if (it != handlers_.end())
            {
                try
                {
                    it->second(payload, payload_size, sequence_id);
                }
                catch (const std::exception &e)
                {
                    std::cerr << "[IPC] Handler exception for signal_type="
                              << signal_type << ": " << e.what() << std::endl;
                    return false;
                }
            }
            else
            {
                std::cerr << "[IPC] No handler for signal_type=" << signal_type << std::endl;
            }

            return true;
        }

    } // namespace ipc
} // namespace aivision
