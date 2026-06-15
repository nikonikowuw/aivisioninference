#ifndef AIVISION_COMMAND_DISPATCHER_H
#define AIVISION_COMMAND_DISPATCHER_H

#include <string>
#include <functional>
#include <memory>
#include <cstdint>

namespace aivision
{
    class InferenceEngine;

    /// 统一指令分发器，用于接收并处理来自 IPC 或 MQTT 的控制命令
    class CommandDispatcher
    {
    public:
        explicit CommandDispatcher(InferenceEngine *engine);
        ~CommandDispatcher() = default;

        /// 处理来自 IPC 的 FlatBuffers/JSON 指令
        void DispatchIPCCommand(uint32_t cmd_type, const uint8_t *payload, size_t size, int client_fd);

        /// 处理来自 MQTT 的 JSON 指令
        /// @param cmd_name 指令名称 (如 "start_stream", "stop_stream", "self_check" 等)
        /// @param payload_json JSON 载荷
        void DispatchMqttCommand(const std::string &cmd_name, const std::string &payload_json);

    private:
        InferenceEngine *engine_;
    };

} // namespace aivision

#endif // AIVISION_COMMAND_DISPATCHER_H
