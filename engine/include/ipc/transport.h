#ifndef AIVISION_IPC_TRANSPORT_H
#define AIVISION_IPC_TRANSPORT_H

// 跨环境的 Transport 接口抽象层。
// 本地使用 UDS，未来分布式可切换至 gRPC/TCP。
// 设计原则：与 FlatBuffers 序列化后的字节流解耦，调用者自行编解码。

#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <vector>

namespace aivision
{
    namespace ipc
    {

        /// 消息回调：接收原始字节流 (FlatBuffers 已序列化数据)
        using MessageCallback =
            std::function<void(const uint8_t *data, size_t size)>;

        /// 连接状态回调
        enum class TransportState
        {
            Disconnected,
            Connecting,
            Connected,
            Reconnecting,
            Failed,
        };

        using StateCallback = std::function<void(TransportState state)>;

        /// Transport 接口：定义跨进程/跨网络通信的抽象
        class Transport
        {
        public:
            virtual ~Transport() = default;

            /// 启动监听 (Server 模式)
            virtual bool StartListen(const std::string &addr) = 0;

            /// 连接到远端 (Client 模式)
            virtual bool Connect(const std::string &addr) = 0;

            /// 断开连接
            virtual void Disconnect() = 0;

            /// 发送消息
            virtual bool Send(const uint8_t *data, size_t size) = 0;

            /// 设置消息接收回调
            virtual void SetMessageCallback(MessageCallback cb) = 0;

            /// 设置连接状态回调
            virtual void SetStateCallback(StateCallback cb) = 0;

            /// 获取当前连接状态
            virtual TransportState GetState() const = 0;

            /// 检查是否已连接
            virtual bool IsConnected() const = 0;

            /// 获取绑定的地址
            virtual std::string GetAddress() const = 0;
        };

        /// Transport 工厂函数
        enum class TransportType
        {
            UDS, // Unix Domain Socket (单机默认)
            TCP, // TCP Socket
        };

        std::unique_ptr<Transport> CreateTransport(TransportType type);

    } // namespace ipc
} // namespace aivision

#endif // AIVISION_IPC_TRANSPORT_H
