#ifndef AIVISION_IPC_SERVER_H
#define AIVISION_IPC_SERVER_H

// C++ 侧 IPC Server — 基于 UDS 的 FlatBuffers 消息服务端。
// 职责：
//   1. 建立 UDS 监听，接受 Go 控制面的连接。
//   2. 接收并反序列化 FlatBuffers 指令 (commands.fbs)。
//   3. 将指令分发到对应 Handler。
//   4. 发送 FlatBuffers 结果 (results.fbs) 回 Go 侧。

#include <atomic>
#include <functional>
#include <memory>
#include <string>
#include <thread>
#include <unordered_map>

#include "transport.h"
#include "proto/flatbuf/envelope_generated.h"

namespace aivision
{
    namespace ipc
    {

        /// 指令处理器函数签名
        using CommandHandler = std::function<void(
            const uint8_t *payload, size_t size, uint64_t sequence_id)>;

        /// IPC Server 配置
        struct IPCServerConfig
        {
            /// UDS Socket 路径
            std::string socket_path = "/tmp/aivision_ipc.sock";
            /// 最大挂起连接数
            int backlog = 32;
            /// 接收缓冲区大小 (字节)
            size_t recv_buffer_size = 65536;
            /// 心跳超时 (秒)
            int heartbeat_timeout_sec = 30;
        };

        /// IPC Server 类
        class IPCServer
        {
        public:
            explicit IPCServer(const IPCServerConfig &config);
            ~IPCServer();

            /// 启动服务 (阻塞)
            bool Start();

            /// 停止服务
            void Stop();

            /// 注册指令 Handler
            void RegisterHandler(uint16_t signal_type, CommandHandler handler);

            /// 获取当前线程正在处理的客户端 fd (线程局部变量)
            static int GetActiveClientFd();

            /// 发送同步响应到指定的客户端 fd
            bool SendResponse(int client_fd, uint32_t resp_type, const uint8_t *payload, size_t payload_len);

            /// 发送结果消息到 Go 侧
            bool SendMessage(uint16_t signal_type,
                             flatbuffers::FlatBufferBuilder &fbb);

            /// 广播结果消息到所有连接的客户端
            bool BroadcastMessage(uint16_t signal_type,
                                  flatbuffers::FlatBufferBuilder &fbb);

            /// 获取配置
            const IPCServerConfig &GetConfig() const { return config_; }

            /// 检查是否运行中
            bool IsRunning() const { return running_.load(); }

        private:
            /// 接受连接循环
            void AcceptLoop();

            /// 处理单个客户端连接
            void HandleClient(int client_fd);

            /// 读取并处理一条消息
            bool ProcessMessage(int client_fd, uint8_t *buffer, size_t size);

            IPCServerConfig config_;
            std::atomic<bool> running_{false};
            int server_fd_{-1};
            std::unique_ptr<std::thread> accept_thread_;

            /// 所有已连接的客户端 fd (用于广播)
            std::vector<int> client_fds_;
            std::mutex clients_mutex_;

            /// 已连接的客户端处理线程 (Stop 时 join)
            std::vector<std::unique_ptr<std::thread>> client_threads_;
            std::mutex threads_mutex_;

            /// 已注册的指令 Handler (独立锁，避免与 clients_mutex_ 交叉死锁)
            std::unordered_map<uint16_t, CommandHandler> handlers_;
            std::mutex handlers_mutex_;

            /// Transport 抽象 (当前使用 UDS)
            std::unique_ptr<Transport> transport_;
        };

    } // namespace ipc
} // namespace aivision

#endif // AIVISION_IPC_SERVER_H
