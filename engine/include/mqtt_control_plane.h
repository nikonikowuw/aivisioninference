#ifndef AIVISION_MQTT_CONTROL_PLANE_H
#define AIVISION_MQTT_CONTROL_PLANE_H

#include <flatbuffers/flatbuffers.h>
#include <memory>
#include <string>
#include <vector>
#include <thread>
#include <asio.hpp>

namespace aivision {
class InferenceEngine;

class MqttControlPlane {
public:
  explicit MqttControlPlane(InferenceEngine *engine);
  ~MqttControlPlane();

  bool Start();
  void Stop();

  bool PublishEvent(uint16_t signal_type, flatbuffers::FlatBufferBuilder &fbb);
  bool PublishResponse(const std::string &cmd_type,
                       const std::string &payload_json);

  bool IsConnected() const;

private:
  InferenceEngine *engine_;
  struct Impl;
  std::unique_ptr<Impl> impl_;

  // Task queue for offloading command execution
  asio::io_context io_context_;
  std::unique_ptr<asio::executor_work_guard<asio::io_context::executor_type>>
      work_guard_;
  std::vector<std::thread> worker_threads_;
};

} // namespace aivision

#endif // AIVISION_MQTT_CONTROL_PLANE_H
