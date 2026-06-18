#ifndef AIVISION_RESPONSE_ROUTER_H
#define AIVISION_RESPONSE_ROUTER_H

// 控制响应路由。
//
// 当前运行时只有 MQTT 控制命令链路：
//   1. Go 通过 MQTT JSON command 下发控制命令。
//   2. MqttControlPlane 将命令交给 CommandDispatcher。
//   3. Handler 内部仍可构造 FlatBuffers 或 JSON 响应。
//   4. ResponseRouter 在 MQTT 上下文中把响应转交给 MQTT response callback。

#include <functional>
#include <string>

namespace aivision {

/// MQTT 上下文 RAII 守护类，确保 thread_local 状态正确清理。
class ScopedMqttContext {
public:
  explicit ScopedMqttContext(const std::string &trace_id);
  ~ScopedMqttContext();
};

/// 负责将内部响应分发到当前控制通道。
class ResponseRouter {
public:
  ResponseRouter();
  ~ResponseRouter();

  /// 获取当前线程正在处理的控制通道标识，-2 表示 MQTT。
  static int GetActiveClientFd();
  static void SetActiveClientFd(int fd);
  static void SetActiveMqttTraceId(const std::string &trace_id);
  static std::string GetActiveMqttTraceId();

  using MqttResponseCallback =
      std::function<void(uint32_t resp_type, const uint8_t *payload,
                         size_t size)>;
  void SetMqttResponseCallback(MqttResponseCallback cb) {
    mqtt_response_callback_ = std::move(cb);
  }

  /// 发送同步响应到当前控制通道。
  bool SendResponse(int client_fd, uint32_t resp_type, const uint8_t *payload,
                    size_t payload_len);

private:
  MqttResponseCallback mqtt_response_callback_;
};

} // namespace aivision

#endif // AIVISION_RESPONSE_ROUTER_H
