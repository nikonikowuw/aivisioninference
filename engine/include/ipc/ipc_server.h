#ifndef AIVISION_IPC_SERVER_H
#define AIVISION_IPC_SERVER_H

// C++ 侧 IPC Server — 基于 TCP 的 FlatBuffers 消息服务端。
// 职责：
//   1. 建立 TCP 监听，接受 Go 控制面的连接。
//   2. 接收并反序列化 FlatBuffers 指令 (commands.fbs)。
//   3. 将指令分发到对应 Handler。
//   4. 发送 FlatBuffers 结果 (results.fbs) 回 Go 侧。

#include <atomic>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>

#include "proto/flatbuf/envelope_generated.h"
#include "transport.h"

namespace aivision {
class CommandDispatcher;

/// MQTT 上下文 RAII 守护类，确保 thread_local 状态正确清理
class ScopedMqttContext {
public:
  explicit ScopedMqttContext(const std::string &trace_id);
  ~ScopedMqttContext();
};

namespace ipc {

/// ResponseRouter 类 (负责将内部响应分发到 MQTT 或其他传输层)
class ResponseRouter {
public:
  explicit ResponseRouter();
  ~ResponseRouter();

  /// 关联指令分发器
  void SetCommandDispatcher(aivision::CommandDispatcher *dispatcher) {
    dispatcher_ = dispatcher;
  }

  /// 获取当前线程正在处理的客户端 fd (线程局部变量，-2 表示 MQTT)
  static int GetActiveClientFd();
  static void SetActiveClientFd(int fd);
  static void SetActiveMqttTraceId(const std::string &trace_id);
  static std::string GetActiveMqttTraceId();

  using MqttResponseCallback = std::function<void(
      uint32_t resp_type, const uint8_t *payload, size_t size)>;
  void SetMqttResponseCallback(MqttResponseCallback cb) {
    mqtt_response_callback_ = std::move(cb);
  }

  /// 发送同步响应到指定的客户端 fd (如果 fd == -2 则转发给 MQTT)
  bool SendResponse(int client_fd, uint32_t resp_type, const uint8_t *payload,
                    size_t payload_len);

  /// 广播结果消息 (已弃用，无操作)
  bool BroadcastMessage(uint16_t signal_type,
                        flatbuffers::FlatBufferBuilder &fbb) { return true; }

  /// 检查是否运行中 (始终返回 true)
  bool IsRunning() const { return true; }

private:
  aivision::CommandDispatcher *dispatcher_{nullptr};
  MqttResponseCallback mqtt_response_callback_;
};

} // namespace ipc
} // namespace aivision

#endif // AIVISION_IPC_SERVER_H
