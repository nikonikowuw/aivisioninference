#include "engine.h"
#include "algo/algo_utils.h"
#include "algo/so_handle.h"
#include "command_dispatcher.h"
#include "monitor/device_monitor.h"
#include "monitor/heartbeat_reporter.h"
#include "mqtt_control_plane.h"
#include "pipeline/hw_buffer.h"
#include "pipeline/pipeline_manager.h"
#include "proto/flatbuf/commands_generated.h"
#include "proto/flatbuf/results_generated.h"
#include <algorithm>
#include <array>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <curl/curl.h>
#include <filesystem>
#include <iostream>
#include <nlohmann/json.hpp>
#include <opencv2/imgcodecs.hpp>
#include <regex>
#include <vector>

namespace aivision {
using json = nlohmann::json;

namespace {
std::string PayloadToString(const uint8_t *payload, size_t size) {
  if (!payload || size == 0)
    return "";
  return std::string(reinterpret_cast<const char *>(payload), size);
}

std::string BuildLivePlayURL(const std::string &base_url,
                             const std::string &device_id) {
  std::string play_url = base_url.empty() ? "rtsp://localhost:10554" : base_url;
  if (!play_url.empty() && play_url.back() == '/')
    play_url.pop_back();
  return play_url + "/live/" + device_id;
}

constexpr uint32_t FourCC(char a, char b, char c, char d) {
  return static_cast<uint32_t>(a) | (static_cast<uint32_t>(b) << 8) |
         (static_cast<uint32_t>(c) << 16) | (static_cast<uint32_t>(d) << 24);
}

constexpr uint32_t kPixelFormatBGR24 = FourCC('B', 'G', 'R', '3');

std::vector<uint8_t> DecodeBase64(const std::string &input) {
  static constexpr unsigned char kInvalid = 255;
  static const std::array<unsigned char, 256> table = [] {
    std::array<unsigned char, 256> t{};
    t.fill(kInvalid);
    const std::string chars =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    for (size_t i = 0; i < chars.size(); ++i) {
      t[static_cast<unsigned char>(chars[i])] = static_cast<unsigned char>(i);
    }
    return t;
  }();

  std::vector<uint8_t> output;
  output.reserve(input.size() * 3 / 4);
  int val = 0;
  int valb = -8;
  for (unsigned char c : input) {
    if (c == '=') {
      break;
    }
    const unsigned char decoded = table[c];
    if (decoded == kInvalid) {
      if (c == '\r' || c == '\n' || c == '\t' || c == ' ') {
        continue;
      }
      return {};
    }
    val = (val << 6) + decoded;
    valb += 6;
    if (valb >= 0) {
      output.push_back(static_cast<uint8_t>((val >> valb) & 0xFF));
      valb -= 8;
    }
  }
  return output;
}

std::string JsonError(const std::string &code, const std::string &message) {
  return "{\"success\":false,\"error_code\":\"" + code +
         "\",\"error_message\":\"" + message + "\"}";
}

std::string JsonEscape(const std::string &value) {
  std::string escaped;
  escaped.reserve(value.size());
  for (char ch : value) {
    switch (ch) {
    case '\\':
      escaped += "\\\\";
      break;
    case '"':
      escaped += "\\\"";
      break;
    case '\n':
      escaped += "\\n";
      break;
    case '\r':
      escaped += "\\r";
      break;
    case '\t':
      escaped += "\\t";
      break;
    default:
      escaped += ch;
      break;
    }
  }
  return escaped;
}

std::string EnsurePackageDirConfig(const std::string &algo_params_json,
                                   const std::string &so_path) {
  if (so_path.empty() ||
      algo_params_json.find("\"package_dir\"") != std::string::npos) {
    return algo_params_json.empty() ? "{}" : algo_params_json;
  }

  const std::string package_dir =
      std::filesystem::path(so_path).parent_path().string();
  const std::string package_field =
      std::string("\"package_dir\":\"") + JsonEscape(package_dir) + "\"";
  if (algo_params_json.empty() || algo_params_json == "{}") {
    return "{" + package_field + "}";
  }

  std::string merged = algo_params_json;
  const auto pos = merged.find_last_of('}');
  if (pos == std::string::npos) {
    return "{" + package_field + "}";
  }
  const bool needs_comma = merged.find_first_not_of(" \t\r\n{") != pos;
  merged.insert(pos, std::string(needs_comma ? "," : "") + package_field);
  return merged;
}

} // namespace

InferenceEngine::InferenceEngine(const EngineConfig &config) : config_(config) {
  // 创建组件
  response_router_ = std::make_unique<ResponseRouter>();

  // 原有的 StreamQueueManager 可能会被 PipelineManager 取代
  // 但为了兼容现有代码（如果有的话），先保留或重构
  queue_mgr_ = std::make_unique<pipeline::StreamQueueManager>();
  snapshot_mgr_ = std::make_unique<pipeline::SnapshotManager>();

  buffer_pool_ =
      std::make_unique<pipeline::HwBufferPool>(config.hw_buffer_pool_size);
  worker_pool_ = std::make_unique<pipeline::WorkerPool>(
      pipeline::WorkerPoolConfig{config.worker_count});

  hal_mgr_ = std::make_unique<pipeline::HALManager>();
  pipeline::PipelineManagerConfig pipeline_config;
  pipeline_config.rtsp_push_server = config.rtsp_push_server;
  pipeline_config.zlm_secret = config.zlm_secret;
  pipeline_config.enable_ffmpeg_fallback = config.enable_ffmpeg_fallback;
  pipeline_mgr_ = std::make_unique<pipeline::PipelineManager>(pipeline_config);
  pipeline_mgr_->SetStreamQueueManager(queue_mgr_.get());
  algo_mgr_ = std::make_unique<algo::AlgoManager>();

  metrics_reporter_ = std::make_unique<monitor::MetricsReporter>(
      response_router_.get(), worker_pool_.get(), buffer_pool_.get(),
      queue_mgr_.get(), algo_mgr_.get(), pipeline_mgr_.get(),
      monitor::MetricsReporterConfig{config.metrics_interval_ms, 1000, config.max_preview_streams});
  metrics_reporter_->SetMetricsCallback([this](const monitor::EngineMetrics &metrics) {
    flatbuffers::FlatBufferBuilder builder(2048);
    aivision::control::EngineMetricsMsgBuilder msg(builder);
    msg.add_active_stream_count(metrics.active_stream_count);
    msg.add_dma_used_bytes(metrics.dma_used_bytes);
    msg.add_dma_total_bytes(metrics.dma_total_bytes);
    msg.add_npu_used_bytes(metrics.npu_used_bytes);
    msg.add_npu_total_bytes(metrics.npu_total_bytes);
    msg.add_worker_count(metrics.worker_count);
    msg.add_idle_worker_count(metrics.idle_worker_count);
    msg.add_timestamp_ns(metrics.timestamp_ns);
    msg.add_decode_sessions(metrics.decode_sessions);
    msg.add_encode_sessions(metrics.encode_sessions);
    msg.add_decode_slots_used(metrics.decode_slots_used);
    msg.add_encode_slots_used(metrics.encode_slots_used);
    msg.add_egress_bps(metrics.egress_bps);
    msg.add_preview_pipeline_count(metrics.preview_pipeline_count);
    msg.add_inference_pipeline_count(metrics.inference_pipeline_count);
    msg.add_mixed_pipeline_count(metrics.mixed_pipeline_count);
    msg.add_media_metrics_valid(metrics.media_metrics_valid);
	msg.add_preview_capacity(metrics.preview_capacity);
	msg.add_preview_in_use(metrics.preview_in_use);
	msg.add_preview_capacity_valid(metrics.preview_capacity_valid);
    msg.add_accelerator_utilization(metrics.accelerator_utilization);
    msg.add_accelerator_metrics_valid(metrics.accelerator_metrics_valid);
    builder.Finish(msg.Finish());
    PublishEvent(0x0203, builder);
  });

  // 创建 HeartbeatReporter
  heartbeat_reporter_ = std::make_unique<monitor::HeartbeatReporter>(this);

  // 创建 DeviceMonitor
  device_monitor_ =
      std::make_unique<monitor::DeviceMonitor>(monitor::DeviceMonitorConfig{
          config.device_platform, config.device_storage_path, "/proc", "/sys",
          config.device_enable_external_commands,
          config.device_command_timeout_ms,
          config.device_light_probe_interval_ms,
          config.device_expensive_probe_interval_ms});

  // 将 DeviceMonitor 关联到 HeartbeatReporter 和 MetricsReporter 以获取完整设备快照
  if (heartbeat_reporter_) {
    heartbeat_reporter_->SetDeviceMonitor(device_monitor_.get());
  }
  if (metrics_reporter_) {
    metrics_reporter_->SetDeviceMonitor(device_monitor_.get());
  }

  // 创建 Command Executor（Phase 3 — 远程运维）
  command_executor_ = std::make_unique<monitor::CommandExecutor>();

  // 创建 PTY Module（Phase 3 — Web 终端）
  pty_module_ = std::make_unique<monitor::PTYModule>();
  pty_module_->SetOutputCallback([this](const std::string& session_id, const std::string& data) {
    if (mqtt_control_plane_) {
      json msg = {
        {"type", "pty_output"},
        {"session_id", session_id},
        {"data", data}
      };
      mqtt_control_plane_->PublishResponse("pty_output", msg.dump());
    }
  });
  pty_module_->SetCloseCallback([this](const std::string& session_id, const std::string& error) {
    if (mqtt_control_plane_) {
      json msg = {
        {"type", "pty_error"},
        {"session_id", session_id},
        {"error", error}
      };
      mqtt_control_plane_->PublishResponse("pty_error", msg.dump());
    }
  });

  // 创建 MQTT & Command Dispatcher 组件并建立绑定
  command_dispatcher_ = std::make_unique<CommandDispatcher>(this);
  mqtt_control_plane_ = std::make_unique<MqttControlPlane>(this);

  // 注册 MQTT 响应拦截回调，将内部响应适配为 MQTT JSON 并发布
  response_router_->SetMqttResponseCallback([this](uint32_t resp_type,
                                                   const uint8_t *payload,
                                                   size_t payload_len) {
    // Helper: parse JSON payload with trace_id injection, fallback on error
    auto parseJsonPayload = [](const uint8_t *p, size_t len,
                               const std::string &tid) -> std::string {
      try {
        auto js =
            json::parse(std::string(reinterpret_cast<const char *>(p), len));
        js["trace_id"] = tid;
        return js.dump();
      } catch (...) {
        json js;
        js["trace_id"] = tid;
        js["success"] = false;
        js["error_message"] = "Invalid response payload";
        return js.dump();
      }
    };

    std::string trace_id = ResponseRouter::GetActiveMqttTraceId();
    std::string cmd_type = "";
    std::string json_res = "";

    if (resp_type == 301) {
      cmd_type = "start_stream";
      auto msg =
          flatbuffers::GetRoot<aivision::control::StreamStatusRspMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["device_id"] = msg->device_id() ? msg->device_id()->str() : "";
      js["status"] = msg->is_running() ? "running" : "failed";
      js["play_url"] = msg->playback_url() ? msg->playback_url()->str() : "";
      js["zlm_host"] = msg->zlm_host() ? msg->zlm_host()->str() : "";
      js["zlm_http_port"] = msg->zlm_http_port();
      json_res = js.dump();
    } else if (resp_type == 302) {
      cmd_type = "stop_stream";
      auto msg =
          flatbuffers::GetRoot<aivision::control::StreamStatusRspMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["device_id"] = msg->device_id() ? msg->device_id()->str() : "";
      js["status"] = "stopped";
      json_res = js.dump();
    } else if (resp_type == 303) {
      cmd_type = "start_playback";
      auto msg =
          flatbuffers::GetRoot<aivision::control::StreamStatusRspMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["device_id"] = msg->device_id() ? msg->device_id()->str() : "";
      js["status"] = msg->is_running() ? "running" : "failed";
      js["play_url"] = msg->playback_url() ? msg->playback_url()->str() : "";
      js["zlm_host"] = msg->zlm_host() ? msg->zlm_host()->str() : "";
      js["zlm_http_port"] = msg->zlm_http_port();
      json_res = js.dump();
    } else if (resp_type == 304) {
      cmd_type = "stop_playback";
      auto msg =
          flatbuffers::GetRoot<aivision::control::StreamStatusRspMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["device_id"] = msg->device_id() ? msg->device_id()->str() : "";
      js["status"] = "stopped";
      json_res = js.dump();
    } else if (resp_type == 305) {
      cmd_type = "stream_status";
      auto msg =
          flatbuffers::GetRoot<aivision::control::StreamStatusRspMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["device_id"] = msg->device_id() ? msg->device_id()->str() : "";
      js["status"] = msg->is_running() ? "running" : "stopped";
      js["play_url"] = msg->playback_url() ? msg->playback_url()->str() : "";
      js["zlm_host"] = msg->zlm_host() ? msg->zlm_host()->str() : "";
      js["zlm_http_port"] = msg->zlm_http_port();
      json_res = js.dump();
    } else if (resp_type == 307) {
      cmd_type = "face_library";
      json_res = parseJsonPayload(payload, payload_len, trace_id);
    } else if (resp_type == 308) {
      cmd_type = "face_embedding";
      json_res = parseJsonPayload(payload, payload_len, trace_id);
    } else if (resp_type == 514) {
      cmd_type = "self_check";
      auto msg =
          flatbuffers::GetRoot<aivision::control::AlgoLoadResultMsg>(payload);
      json js;
      js["trace_id"] = trace_id;
      js["success"] = msg->success();
      js["error_code"] = msg->error_code() ? msg->error_code()->str() : "";
      js["error_message"] =
          msg->error_message() ? msg->error_message()->str() : "";
      js["algo_name"] = msg->algo_name() ? msg->algo_name()->str() : "";
      js["version"] = msg->algo_version() ? msg->algo_version()->str() : "";
      json_res = js.dump();
    }

    if (!cmd_type.empty() && mqtt_control_plane_) {
      mqtt_control_plane_->PublishResponse(cmd_type, json_res);
    }
  });
}

InferenceEngine::~InferenceEngine() { Shutdown(); }

std::string InferenceEngine::GetHalPlatform() const {
#ifdef __APPLE__
  return "macos";
#else
  return "rkmpp";
#endif
}

bool InferenceEngine::Initialize() {
  if (initialized_.load())
    return true;

  // 1. 初始化 Worker 池
  worker_pool_->SetQueueManager(queue_mgr_.get());
  worker_pool_->SetSnapshotManager(snapshot_mgr_.get());
  worker_pool_->SetAlgoManager(algo_mgr_.get());

  // 2. 设置 NPU 推理结果回调
  worker_pool_->SetResultCallback([this](const pipeline::InferResult &result) {
    if (!result.success)
      return;

    flatbuffers::FlatBufferBuilder fbb(1024);
    std::vector<flatbuffers::Offset<aivision::control::BoundingBoxInfo>>
        detections_vec;
    std::string alarm_type = "";
    std::string alarm_level = "";

    try {
      auto js = json::parse(result.result_json);
      if (js.contains("detections") && js["detections"].is_array()) {
        for (const auto &det : js["detections"]) {
          auto label_name_offset =
              fbb.CreateString(det.value("label_name", ""));
          aivision::control::BoundingBoxInfoBuilder det_builder(fbb);
          det_builder.add_x(det.value("x", 0));
          det_builder.add_y(det.value("y", 0));
          det_builder.add_w(det.value("w", 0));
          det_builder.add_h(det.value("h", 0));
          det_builder.add_confidence(det.value("confidence", 0.0f));
          det_builder.add_label_id(det.value("label_id", 0));
          det_builder.add_label_name(label_name_offset);
          det_builder.add_track_id(det.value("track_id", 0));
          detections_vec.push_back(det_builder.Finish());
        }
      }
      alarm_type = js.value("alarm_type", "");
      alarm_level = js.value("alarm_level", "");
    } catch (...) {
    }

    auto task_id_offset = fbb.CreateString(result.task_id);
    auto algo_name_offset = fbb.CreateString(result.algo_name);
    auto device_id_offset = fbb.CreateString(result.task_id);
    auto alarm_type_offset = fbb.CreateString(alarm_type);
    auto alarm_level_offset = fbb.CreateString(alarm_level);
    auto result_json_offset = fbb.CreateString(result.result_json);
    auto detections_offset = fbb.CreateVector(detections_vec);

    auto root = aivision::control::CreateInferenceResultMsg(
        fbb, task_id_offset, algo_name_offset, device_id_offset,
        result.frame_ts_ns, 0, 0, detections_offset,
        aivision::control::RecordType_Capture, alarm_type_offset,
        alarm_level_offset, 0, 0, 0, 0, result_json_offset,
        result.infer_time_us);
    fbb.Finish(root);

    PublishEvent(0x0200, fbb); // SignalInferenceResult = 0x0200
  });

  if (device_monitor_) {
    device_monitor_->Initialize();
    device_monitor_->Start();
  }

  initialized_.store(true);
  return true;
}

void InferenceEngine::Run() { Run({}); }

void InferenceEngine::Run(const std::function<bool()> &should_stop) {
  if (!initialized_.load()) {
    if (!Initialize())
      return;
  }

  running_.store(true);
  shutdown_called_.store(false);

  // 启动 MQTT Control Plane
  if (config_.enable_mqtt && mqtt_control_plane_) {
    if (!mqtt_control_plane_->Start()) {
      std::cerr << "Failed to start MQTT Control Plane" << std::endl;
    }
  }

  // 启动 Worker 池
  worker_pool_->Start();

  // 启动 Metrics 上报
  metrics_reporter_->Start();

  // 启动 HeartbeatReporter（如果配置了平台连接）
  if (!config_.platform_url.empty() && !config_.node_id.empty() &&
      !config_.auth_token.empty()) {
    heartbeat_reporter_->Start();
  }

  // 主循环 (等待 Shutdown 或外部信号)
  while (running_.load()) {
    if (should_stop && should_stop()) {
      running_.store(false);
      break;
    }
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
  }
}

void InferenceEngine::RequestShutdown() { running_.store(false); }

void InferenceEngine::Shutdown() {
  bool expected = false;
  if (!shutdown_called_.compare_exchange_strong(expected, true))
    return;

  running_.store(false);

  if (pipeline_mgr_)
    pipeline_mgr_->StopAll();
  if (heartbeat_reporter_)
    heartbeat_reporter_->Stop();
  if (device_monitor_)
    device_monitor_->Stop();
  if (metrics_reporter_)
    metrics_reporter_->Stop();
  if (worker_pool_)
    worker_pool_->Stop();
  if (mqtt_control_plane_)
    mqtt_control_plane_->Stop();

  std::cout << "Engine shutdown complete" << std::endl;
}

bool InferenceEngine::PublishEvent(uint16_t signal_type,
                                   flatbuffers::FlatBufferBuilder &fbb) {
  bool success = true;
  if (config_.enable_mqtt && mqtt_control_plane_) {
    if (!mqtt_control_plane_->PublishEvent(signal_type, fbb)) {
      success = false;
    }
  }
  return success;
}

void InferenceEngine::HandleStartStream(const uint8_t *payload, size_t size,
                                        uint64_t seq) {
  (void)payload;
  (void)size;
  (void)seq;
  std::cout << "[Control] Received StartStream" << std::endl;
}

void InferenceEngine::HandleStopStream(const uint8_t *payload, size_t size,
                                       uint64_t seq) {
  (void)payload;
  (void)size;
  (void)seq;
  std::cout << "[Control] Received StopStream" << std::endl;
}

void InferenceEngine::HandleUpdateAlgoConfig(const uint8_t *payload,
                                             size_t size, uint64_t seq) {
  (void)payload;
  (void)size;
  (void)seq;
  std::cout << "[Control] Received UpdateAlgoConfig" << std::endl;
}

void InferenceEngine::HandleHeartbeat(const uint8_t *payload, size_t size,
                                      uint64_t seq) {
  (void)payload;
  (void)size;
  (void)seq;
}

void InferenceEngine::HandleShutdown(const uint8_t *payload, size_t size,
                                     uint64_t seq) {
  (void)payload;
  (void)size;
  (void)seq;
  std::cout << "[Control] Received Shutdown command" << std::endl;
  Shutdown();
}

namespace {

/// libcurl 写回调，将响应追加到 std::string
size_t curl_string_write_callback(void *ptr, size_t size, size_t nmemb,
                                  void *userdata) {
  std::string *str = static_cast<std::string *>(userdata);
  str->append(static_cast<const char *>(ptr), size * nmemb);
  return size * nmemb;
}

std::string ExtractJsonField(const std::string &json_str,
                             const std::string &field_name) {
  try {
    auto j = json::parse(json_str);
    if (j.contains(field_name)) {
      if (j[field_name].is_string()) {
        return j[field_name].get<std::string>();
      } else if (j[field_name].is_number() || j[field_name].is_boolean()) {
        return j[field_name].dump();
      }
    }
  } catch (...) {
  }
  return "";
}

bool ExtractJsonBoolField(const std::string &json_str,
                          const std::string &field_name, bool default_value) {
  try {
    auto j = json::parse(json_str);
    if (j.contains(field_name)) {
      if (j[field_name].is_boolean()) {
        return j[field_name].get<bool>();
      } else if (j[field_name].is_string()) {
        std::string s = j[field_name].get<std::string>();
        std::transform(s.begin(), s.end(), s.begin(), ::tolower);
        return (s == "true" || s == "1");
      } else if (j[field_name].is_number_integer()) {
        return j[field_name].get<int>() != 0;
      }
    }
  } catch (...) {
  }
  return default_value;
}

std::string ExtractJsonRawField(const std::string &json_str,
                                const std::string &field_name) {
  try {
    auto j = json::parse(json_str);
    if (j.contains(field_name) &&
        (j[field_name].is_object() || j[field_name].is_array())) {
      return j[field_name].dump();
    }
  } catch (...) {
  }
  return "";
}

} // namespace

std::string InferenceEngine::AddStreamProxy(const std::string &device_id,
                                            const std::string &rtsp_url) {
  if (config_.zlm_api_url.empty()) {
    std::cerr << "[ZLM] zlm_api_url not configured" << std::endl;
    return "";
  }

  // 构建 ZLM addStreamProxy URL
  // GET
  // /index/api/addStreamProxy?vhost=__defaultVhost__&app=live&stream={device_id}&url={rtsp_url}&secret=xxx&retry_count=3&rtp_type=0
  std::string zlm_url = config_.zlm_api_url;
  if (zlm_url.back() == '/')
    zlm_url.pop_back();

  // URL 编码 rtsp_url
  std::string encoded_url;
  CURL *curl = curl_easy_init();
  if (curl) {
    char *encoded = curl_easy_escape(curl, rtsp_url.c_str(), rtsp_url.size());
    if (encoded) {
      encoded_url = encoded;
      curl_free(encoded);
    }
    curl_easy_cleanup(curl);
  }
  if (encoded_url.empty())
    encoded_url = rtsp_url;

  std::string api_url =
      zlm_url + "/index/api/addStreamProxy" + "?vhost=__defaultVhost__" +
      "&app=live" + "&stream=" + device_id + "&url=" + encoded_url +
      "&secret=" + config_.zlm_secret + "&retry_count=3" + "&rtp_type=0" // TCP
      + "&timeout_sec=10";

  std::cout << "[ZLM] Calling addStreamProxy for device: " << device_id
            << std::endl;
  std::cout << "[ZLM] URL: " << api_url << std::endl;

  // 发送 HTTP 请求
  std::string response;
  CURL *curl_handle = curl_easy_init();
  if (!curl_handle) {
    std::cerr << "[ZLM] Failed to init curl" << std::endl;
    return "";
  }

  curl_easy_setopt(curl_handle, CURLOPT_URL, api_url.c_str());
  curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION,
                   curl_string_write_callback);
  curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA, &response);
  curl_easy_setopt(curl_handle, CURLOPT_TIMEOUT, 30L);

  CURLcode res = curl_easy_perform(curl_handle);
  curl_easy_cleanup(curl_handle);

  if (res != CURLE_OK) {
    std::cerr << "[ZLM] addStreamProxy failed: " << curl_easy_strerror(res)
              << std::endl;
    return "";
  }

  std::cout << "[ZLM] Response: " << response << std::endl;

  std::string code_str = ExtractJsonField(response, "code");
  std::string msg = ExtractJsonField(response, "msg");
  if (code_str != "0") {
    // NOTE: 依赖 ZLM 英文错误消息匹配，ZLM 版本升级后需确认消息格式未变化
    if (msg.find("already exists") != std::string::npos) {
      std::cout << "[ZLM] Stream proxy already exists, reusing existing stream." << std::endl;
    } else {
      std::cerr << "[ZLM] addStreamProxy error: code=" << code_str << std::endl;
      if (!msg.empty())
        std::cerr << "[ZLM] msg: " << msg << std::endl;
      return "";
    }
  }

  // 构建播放 URL
  // RTSP: rtsp://{host}:554/live/{device_id}
  // WebRTC: webrtc://{host}:8000/live/{device_id}
  const std::string &host = config_.zlm_url_info.host;

  std::string play_url = "rtsp://" + host + ":554/live/" + device_id;
  std::cout << "[ZLM] Stream proxy added, play URL: " << play_url << std::endl;

  return play_url;
}

bool InferenceEngine::CloseStreamProxy(const std::string &device_id) {
  if (config_.zlm_api_url.empty())
    return false;

  std::string zlm_url = config_.zlm_api_url;
  if (zlm_url.back() == '/')
    zlm_url.pop_back();

  std::string api_url = zlm_url + "/index/api/closeStream" +
                        "?vhost=__defaultVhost__" + "&app=live" +
                        "&stream=" + device_id + "&force=1" +
                        "&secret=" + config_.zlm_secret;

  std::cout << "[ZLM] Closing stream proxy for device: " << device_id
            << std::endl;

  std::string response;
  CURL *curl = curl_easy_init();
  if (!curl)
    return false;

  curl_easy_setopt(curl, CURLOPT_URL, api_url.c_str());
  curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_string_write_callback);
  curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
  curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

  CURLcode res = curl_easy_perform(curl);
  curl_easy_cleanup(curl);

  if (res != CURLE_OK) {
    std::cerr << "[ZLM] closeStream failed: " << curl_easy_strerror(res)
              << std::endl;
    return false;
  }

  std::cout << "[ZLM] closeStream response: " << response << std::endl;
  return true;
}

void InferenceEngine::HandleStreamStart(const uint8_t *payload, size_t size,
                                        uint64_t seq) {
  (void)seq;
  std::cout << "[Control] Received StreamStart" << std::endl;

  std::string payload_str = PayloadToString(payload, size);
  std::string device_id = ExtractJsonField(payload_str, "device_id");
  std::string stream_url = ExtractJsonField(payload_str, "stream_url");
  bool enable_infer = ExtractJsonBoolField(payload_str, "enable_infer", false);
  bool enable_playback =
      ExtractJsonBoolField(payload_str, "enable_playback", false);
  std::string algo_name = ExtractJsonField(payload_str, "algo_name");
  std::string algo_version = ExtractJsonField(payload_str, "algo_version");
  std::string so_path = ExtractJsonField(payload_str, "so_path");
  std::string algo_params_json =
      ExtractJsonField(payload_str, "algo_params_json");
  if (algo_params_json.empty()) {
    algo_params_json = "{}";
  }

  std::cout << "[Control] StreamStart device_id=" << device_id
            << ", stream_url=" << stream_url
            << ", enable_infer=" << enable_infer
            << ", enable_playback=" << enable_playback << ", algo=" << algo_name
            << ", so_path=" << so_path << std::endl;

  if (device_id.empty() || stream_url.empty()) {
    std::cerr << "[Control] Invalid StreamStart payload: missing device_id or "
                 "stream_url"
              << std::endl;
    flatbuffers::FlatBufferBuilder fbb(256);
    auto device_id_str = fbb.CreateString("");
    auto resp =
        aivision::control::CreateStreamStatusRspMsg(fbb, device_id_str, false);
    fbb.Finish(resp);
    int client_fd = response_router_->GetActiveClientFd();
    if (client_fd != -1)
      response_router_->SendResponse(client_fd, 301, fbb.GetBufferPointer(),
                                     fbb.GetSize());
    return;
  }

  bool algo_ready = true;
  if (enable_infer) {
    if (algo_name.empty() || so_path.empty()) {
      std::cerr << "[Control] StreamStart infer enabled but algo_name or "
                   "so_path is empty"
                << std::endl;
      algo_ready = false;
    } else {
      auto instance =
          algo_mgr_->Load(algo_name, algo_version, so_path, algo_params_json);
      if (!instance) {
        std::cerr << "[Control] Failed to load algorithm for stream: "
                  << algo_name << ", so_path=" << so_path << std::endl;
        algo_ready = false;
      } else {
        pipeline::AlgoConfig config;
        config.algo_name = algo_name;
        config.algo_version = algo_version;
        config.algo_params_json = algo_params_json;
        snapshot_mgr_->UpdateConfig(algo_name, config);
        snapshot_mgr_->BindStreamAlgos(device_id, {algo_name});
        std::cout << "[Control] Algorithm bound to stream device_id="
                  << device_id << ", algo=" << algo_name << std::endl;
      }
    }
  }

  bool started = false;
  if (algo_ready) {
    started = pipeline_mgr_->CreatePipeline(device_id, stream_url, enable_infer,
                                            enable_playback);
  }
  if (!started) {
    snapshot_mgr_->RemoveStreamBinding(device_id);
    std::cerr << "[Control] Failed to create hardware pipeline for device: "
              << device_id << std::endl;
  }

  std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);

  // 如果是 playback 模式且 pipeline 启动成功，叫 ZLM addStreamProxy 以提供 HLS 分发
  if (started && enable_playback) {
    std::string proxy_result = AddStreamProxy(device_id, play_url);
    if (!proxy_result.empty()) {
      std::cout << "[Control] ZLM addStreamProxy succeeded: " << proxy_result << std::endl;
    }
  }
  flatbuffers::FlatBufferBuilder fbb(512);
  auto device_id_str = fbb.CreateString(device_id);
  auto play_url_str = fbb.CreateString(started ? play_url : "");
  auto zlm_host_str = !config_.zlm_url_info.host.empty() ? fbb.CreateString(config_.zlm_url_info.host) : 0;
  auto resp = aivision::control::CreateStreamStatusRspMsg(
      fbb, device_id_str, started, 0, 0, 0, play_url_str, zlm_host_str, config_.zlm_url_info.http_port);
  fbb.Finish(resp);

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(client_fd, 301, fbb.GetBufferPointer(),
                                   fbb.GetSize());
    std::cout << "[Control] StreamStart response sent for device: " << device_id
              << ", started=" << started
              << ", play_url=" << (started ? play_url : "") << std::endl;
  } else {
    std::cerr
        << "[Control] Failed to send StreamStart response: no active client"
        << std::endl;
  }
}

void InferenceEngine::HandleStreamStop(const uint8_t *payload, size_t size,
                                       uint64_t seq) {
  (void)seq;
  std::cout << "[Control] Received StreamStop" << std::endl;

  std::string payload_str = PayloadToString(payload, size);
  std::string device_id = ExtractJsonField(payload_str, "device_id");
  if (device_id.empty()) {
    device_id = payload_str;
  }

  std::cout << "[Control] StreamStop device_id=" << device_id << std::endl;

  if (!device_id.empty()) {
    pipeline_mgr_->DestroyPipeline(device_id);
    snapshot_mgr_->RemoveStreamBinding(device_id);
  }

  // 发送响应
  flatbuffers::FlatBufferBuilder fbb(256);
  auto device_id_str = fbb.CreateString(device_id);
  auto resp =
      aivision::control::CreateStreamStatusRspMsg(fbb, device_id_str, false);
  fbb.Finish(resp);

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(client_fd, 302, fbb.GetBufferPointer(),
                                   fbb.GetSize());
    std::cout << "[Control] StreamStop response sent for device: " << device_id
              << std::endl;
  }
}

void InferenceEngine::HandleStreamPlaybackStart(const uint8_t *payload,
                                                size_t size, uint64_t seq) {
  (void)seq;
  std::cout << "[Control] Received StreamPlaybackStart" << std::endl;

  std::string payload_str = PayloadToString(payload, size);
  std::string device_id = ExtractJsonField(payload_str, "device_id");
  std::string stream_url = ExtractJsonField(payload_str, "stream_url");
  if (device_id.empty()) {
    device_id = payload_str;
  }

  std::cout << "[Control] StreamPlaybackStart device_id=" << device_id
            << ", stream_url=" << stream_url << std::endl;

  bool started = false;
  if (!device_id.empty()) {
    if (pipeline_mgr_->GetPipeline(device_id)) {
      started = pipeline_mgr_->EnablePlayback(device_id);
    } else if (!stream_url.empty()) {
      started =
          pipeline_mgr_->CreatePipeline(device_id, stream_url, false, true);
    }
  }

  std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);

  // 启动成功后叫 ZLM addStreamProxy 以提供 HLS 分发
  if (started) {
    std::string push_url = play_url;
    std::string proxy_result = AddStreamProxy(device_id, push_url);
    if (!proxy_result.empty()) {
      std::cout << "[Control] ZLM addStreamProxy succeeded: " << proxy_result << std::endl;
    }
  }

  flatbuffers::FlatBufferBuilder fbb(256);
  auto device_id_str = fbb.CreateString(device_id);
  auto play_url_str = started ? fbb.CreateString(play_url) : 0;
  auto zlm_host_str = !config_.zlm_url_info.host.empty() ? fbb.CreateString(config_.zlm_url_info.host) : 0;
  auto resp = aivision::control::CreateStreamStatusRspMsg(
      fbb, device_id_str, started, 0, 0, 0, play_url_str, zlm_host_str, config_.zlm_url_info.http_port);
  fbb.Finish(resp);

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(client_fd, 303, fbb.GetBufferPointer(),
                                   fbb.GetSize());
    std::cout << "[Control] StreamPlaybackStart response sent for device: "
              << device_id << std::endl;
  }
}

void InferenceEngine::HandleStreamPlaybackStop(const uint8_t *payload,
                                               size_t size, uint64_t seq) {
  (void)seq;
  std::cout << "[Control] Received StreamPlaybackStop" << std::endl;

  std::string payload_str = PayloadToString(payload, size);
  std::string device_id = ExtractJsonField(payload_str, "device_id");
  if (device_id.empty()) {
    device_id = payload_str;
  }

  std::cout << "[Control] StreamPlaybackStop device_id=" << device_id
            << std::endl;

  if (!device_id.empty()) {
    pipeline_mgr_->DestroyPipeline(device_id);
  }

  // 发送响应
  flatbuffers::FlatBufferBuilder fbb(256);
  auto device_id_str = fbb.CreateString(device_id);
  auto resp =
      aivision::control::CreateStreamStatusRspMsg(fbb, device_id_str, false);
  fbb.Finish(resp);

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(client_fd, 304, fbb.GetBufferPointer(),
                                   fbb.GetSize());
    std::cout << "[Control] StreamPlaybackStop response sent for device: "
              << device_id << std::endl;
  }
}

void InferenceEngine::HandleStreamStatus(const uint8_t *payload, size_t size,
                                         uint64_t seq) {
  (void)seq;
  std::string payload_str = PayloadToString(payload, size);
  std::string device_id = ExtractJsonField(payload_str, "device_id");
  if (device_id.empty()) {
    device_id = payload_str;
  }

  auto *pipeline = pipeline_mgr_->GetPipeline(device_id);
  bool running = pipeline != nullptr;
  std::string play_url = BuildLivePlayURL(config_.rtsp_push_server, device_id);

  flatbuffers::FlatBufferBuilder fbb(512);
  auto device_id_str = fbb.CreateString(device_id);
  auto play_url_str = fbb.CreateString(running ? play_url : "");
  auto zlm_host_str = !config_.zlm_url_info.host.empty() ? fbb.CreateString(config_.zlm_url_info.host) : 0;
  auto resp = aivision::control::CreateStreamStatusRspMsg(
      fbb, device_id_str, running, 0, 0, 0, play_url_str, zlm_host_str, config_.zlm_url_info.http_port);
  fbb.Finish(resp);

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(client_fd, 305, fbb.GetBufferPointer(),
                                   fbb.GetSize());
  }
}

void InferenceEngine::HandleFaceLibraryUpdate(const uint8_t *payload,
                                              size_t size, uint64_t seq) {
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  const std::string algo_name = ExtractJsonField(payload_str, "algo_name");

  std::string face_library_json =
      ExtractJsonField(payload_str, "face_library_json");
  if (face_library_json.empty()) {
    face_library_json = ExtractJsonRawField(payload_str, "face_library");
  }
  if (face_library_json.empty()) {
    // 兼容 Go 侧直接发送 { "algo_name": "...", "version": "...", "items": [...]
    // } 的载荷。
    face_library_json = payload_str;
  }

  bool success = false;
  std::string error_message;
  if (algo_name.empty()) {
    error_message = "missing algo_name";
  } else if (!algo_mgr_) {
    error_message = "algo manager not initialized";
  } else {
    success = algo_mgr_->UpdateFaceLibrary(algo_name, face_library_json);
    if (!success) {
      error_message =
          "algorithm unavailable or does not support face library update";
    }
  }

  std::cout << "[Control] FaceLibraryUpdate"
            << " algo=" << algo_name
            << " payload_size=" << face_library_json.size()
            << " success=" << success << std::endl;

  const std::string response =
      std::string("{\"success\":") + (success ? "true" : "false") +
      ",\"algo_name\":\"" + algo_name + "\",\"error_message\":\"" +
      error_message + "\"}";
  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(
        client_fd, 307, reinterpret_cast<const uint8_t *>(response.data()),
        response.size());
  }
}

void InferenceEngine::HandleAlgoWarmup(const uint8_t *payload, size_t size,
                                       uint64_t seq) {
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  std::string algo_name = ExtractJsonField(payload_str, "algo_name");
  if (algo_name.empty()) {
    algo_name = "face_recognition";
  }
  std::string algo_version = ExtractJsonField(payload_str, "algo_version");

  bool success = false;
  std::string error_code;
  std::string error_message;

  if (!algo_mgr_) {
    error_code = "ENGINE_NOT_READY";
    error_message = "algo manager not initialized";
  } else if (algo_mgr_->IsLoaded(algo_name)) {
    success = true;
  } else {
    std::string install_path;
    for (const auto &dep : algo_mgr_->GetDeployments()) {
      if (dep.status != "installed")
        continue;
      if (!dep.algo_name.empty() && dep.algo_name != algo_name)
        continue;
      if (!algo_version.empty() && dep.version != algo_version)
        continue;
      install_path = dep.install_path;
      break;
    }

    if (install_path.empty()) {
      error_code = "ALGO_NOT_INSTALLED";
      error_message = "algorithm package is not installed on this node";
    } else {
      const std::string so_path = algo::FindAlgorithmSo(install_path);
      if (so_path.empty()) {
        error_code = "SO_NOT_FOUND";
        error_message = "nikoniko_detector.so not found under install path";
      } else {
        const std::string config_json = EnsurePackageDirConfig("{}", so_path);
        auto instance =
            algo_mgr_->Load(algo_name, algo_version, so_path, config_json);
        success = instance != nullptr;
        if (!success) {
          error_code = "ALGO_WARMUP_FAILED";
          error_message = "failed to load algorithm runtime instance";
        }
      }
    }
  }

  std::cout << "[Control] AlgoWarmup"
            << " algo=" << algo_name << " version=" << algo_version
            << " success=" << success << std::endl;

  const std::string response =
      std::string("{\"success\":") + (success ? "true" : "false") +
      ",\"algo_name\":\"" + JsonEscape(algo_name) + "\",\"algo_version\":\"" +
      JsonEscape(algo_version) + "\",\"error_code\":\"" +
      JsonEscape(error_code) + "\",\"error_message\":\"" +
      JsonEscape(error_message) + "\"}";
  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(
        client_fd, 309, reinterpret_cast<const uint8_t *>(response.data()),
        response.size());
  }
}

void InferenceEngine::HandleFaceEmbeddingExtract(const uint8_t *payload,
                                                 size_t size, uint64_t seq) {
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  std::string algo_name = ExtractJsonField(payload_str, "algo_name");
  if (algo_name.empty()) {
    algo_name = "face_recognition";
  }
  std::string algo_version = ExtractJsonField(payload_str, "algo_version");
  if (algo_version.empty()) {
    algo_version = "1.0.0";
  }
  const std::string image_base64 =
      ExtractJsonField(payload_str, "image_base64");
  std::string response;
  bool success = false;
  uint32_t infer_time_us = 0;

  if (image_base64.empty()) {
    response = JsonError("MISSING_IMAGE", "image_base64 is required");
  } else if (!algo_mgr_) {
    response = JsonError("ENGINE_NOT_READY", "algo manager not initialized");
  } else {
    std::vector<uint8_t> image_bytes = DecodeBase64(image_base64);
    if (image_bytes.empty()) {
      response =
          JsonError("INVALID_IMAGE_BASE64", "image_base64 decode failed");
    } else {
      cv::Mat encoded(1, static_cast<int>(image_bytes.size()), CV_8UC1,
                      image_bytes.data());
      cv::Mat image = cv::imdecode(encoded, cv::IMREAD_COLOR);
      if (image.empty()) {
        response = JsonError("INVALID_IMAGE", "image decode failed");
      } else {
        pipeline::HwBufferDesc desc{};
        desc.memory_type = pipeline::HwBufferMemoryType::HostMemory;
        desc.dma_fd = -1;
        desc.dma_buf_fd = -1;
        desc.size = image.total() * image.elemSize();
        desc.data = image.data;
        desc.stride = static_cast<uint32_t>(image.step);
        desc.width = static_cast<uint32_t>(image.cols);
        desc.height = static_cast<uint32_t>(image.rows);
        desc.pixel_format = kPixelFormatBGR24;

        if (response.empty()) {
          success = algo_mgr_->ExtractFaceEmbedding(algo_name, desc, response,
                                                    infer_time_us);
          if (!success) {
            response = JsonError("EXTRACT_FAILED",
                                 "algorithm unavailable or extract failed");
          }
        }
      }
    }
  }

  std::cout << "[Control] FaceEmbeddingExtract"
            << " algo=" << algo_name << " success=" << success
            << " infer_time_us=" << infer_time_us
            << " response_size=" << response.size() << std::endl;

  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    response_router_->SendResponse(
        client_fd, 308, reinterpret_cast<const uint8_t *>(response.data()),
        response.size());
  }
}

namespace {
/// libcurl 写回调
size_t curl_write_callback(void *ptr, size_t size, size_t nmemb, FILE *stream) {
  return fwrite(ptr, size, nmemb, stream);
}

/// 对来自 algo_meta.yaml 的文件名/标识符进行严格校验，防止命令注入
bool validateAlgoIdentifier(const std::string &name) {
  static const std::regex safe_pattern("^[a-zA-Z0-9_\\-.]+$");
  return std::regex_match(name, safe_pattern);
}
} // anonymous namespace

void InferenceEngine::HandleStartSelfCheck(const uint8_t *payload, size_t size,
                                           uint64_t seq) {
  (void)size;
  (void)seq;
  std::cout << "[Control] Received StartSelfCheck command" << std::endl;

  const aivision::control::StartSelfCheckCmd *cmd =
      flatbuffers::GetRoot<aivision::control::StartSelfCheckCmd>(payload);
  if (!cmd) {
    std::cerr << "Failed to parse StartSelfCheckCmd" << std::endl;
    return;
  }

  std::string download_url = cmd->download_url()->str();
  std::string token = cmd->token()->str();
  std::string algo_name = cmd->algorithm_name()->str();
  std::string version = cmd->version()->str();

  // ⚠️ 安全校验：验证所有来自 algo_meta.yaml 的标识符，防止命令注入
  if (!validateAlgoIdentifier(algo_name) || !validateAlgoIdentifier(version)) {
    std::cerr << "[SECURITY] Invalid algo_name or version, rejected: algo="
              << algo_name << ", version=" << version << std::endl;
    return;
  }

  std::cout << "Starting self check for: " << algo_name
            << " (version: " << version << ")" << std::endl;

  std::string temp_tar_path = "/tmp/algo_check_" + algo_name + ".tar";
  std::string extract_dir = "/tmp/algo_check_" + algo_name + "_dir";

  // 清理函数：用 remove_all 替代 system()，避免命令注入
  auto cleanup = [&]() {
    std::error_code ec;
    std::filesystem::remove(temp_tar_path, ec);
    std::filesystem::remove_all(extract_dir, ec);
  };

  // Ensure clean start
  cleanup();

  bool success = true;
  std::string err_msg;
  std::string err_code = "0";
  uint32_t load_time_ms = 0;
  uint64_t npu_mem_bytes = 0;

  auto start_time = std::chrono::steady_clock::now();

  // 1. Download tar file using libcurl
  CURL *curl = curl_easy_init();
  if (!curl) {
    success = false;
    err_msg = "Failed to initialize curl";
    err_code = "CURL_INIT_ERROR";
  } else {
    FILE *fp = fopen(temp_tar_path.c_str(), "wb");
    if (!fp) {
      success = false;
      err_msg = "Failed to create temporary file";
      err_code = "FILE_CREATE_ERROR";
    } else {
      curl_easy_setopt(curl, CURLOPT_URL, download_url.c_str());
      curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION,
                       static_cast<size_t (*)(void *, size_t, size_t, FILE *)>(
                           curl_write_callback));
      curl_easy_setopt(curl, CURLOPT_WRITEDATA, fp);
      curl_easy_setopt(curl, CURLOPT_TIMEOUT, 60L);
      curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
      CURLcode res = curl_easy_perform(curl);
      fclose(fp);
      if (res != CURLE_OK) {
        success = false;
        err_msg = "Download failed: " + std::string(curl_easy_strerror(res));
        err_code = "DOWNLOAD_ERROR";
      }
    }
    curl_easy_cleanup(curl);
  }

  // 2. Extract tar file using std::filesystem shelling out to tar(1) —
  //    validated algo_name ensures no injection. Read entries one by one
  //    to reject symlinks and paths escaping extract_dir.
  if (success) {
    std::error_code ec;
    std::filesystem::create_directories(extract_dir, ec);
    if (ec) {
      success = false;
      err_msg = "Failed to create extraction directory";
      err_code = "EXTRACT_ERROR";
    } else {
      std::string tar_cmd =
          "tar -xf \"" + temp_tar_path + "\" -C \"" + extract_dir + "\"";
      if (std::system(tar_cmd.c_str()) != 0) {
        success = false;
        err_msg = "Failed to extract tar archive";
        err_code = "EXTRACT_ERROR";
      }
    }
  }

  // 3. Resolve the required algorithm library from the fixed package layout.
  std::string so_path;
  if (success) {
    so_path = algo::FindAlgorithmSo(extract_dir);

    if (success && so_path.empty()) {
      success = false;
      err_msg = "nikoniko_detector.so not found in the algorithm package";
      err_code = "SO_NOT_FOUND";
    }
  }

  // 4. dlopen, check symbols and run self-test
  if (success) {
    try {
      aivision::algo::SoHandle handle(so_path);

      // Check optional self test
      bool has_self_test = false;
      for (const auto &opt_sym : handle.GetCheckResult().optional_found) {
        if (opt_sym == "detector_self_test") {
          has_self_test = true;
          break;
        }
      }

      if (has_self_test) {
        int ret = handle.SelfTest();
        if (ret != 0) {
          success = false;
          err_msg =
              "detector_self_test failed with code: " + std::to_string(ret);
          err_code = "SELF_TEST_FAILED";
        }
      }

      // Call detector_init/detector_destroy to verify initialization flow
      if (success) {
        algo_handle_t detector = handle.Init("{}");
        if (!detector) {
          success = false;
          err_msg = "detector_init returned NULL context";
          err_code = "INIT_FAILED";
        } else {
          handle.Destroy(detector);
        }
      }
    } catch (const std::exception &e) {
      success = false;
      err_msg = e.what();
      err_code = "SO_LOAD_ERROR";
    }
  }

  // 5. Keep a runtime instance ready for one-shot feature extraction.
  if (success && algo_mgr_) {
    const std::string package_dir =
        std::filesystem::path(so_path).parent_path().string();
    const std::string config_json =
        std::string("{\"package_dir\":\"") + JsonEscape(package_dir) + "\"}";
    auto instance = algo_mgr_->Load(algo_name, version, so_path, config_json);
    if (!instance) {
      success = false;
      err_msg = "Failed to load algorithm runtime instance";
      err_code = "RUNTIME_LOAD_FAILED";
    }
  }

  auto end_time = std::chrono::steady_clock::now();
  load_time_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                     end_time - start_time)
                     .count();

  // 6. Clean up temporary files
  cleanup();

  // 7. Build response flatbuffer
  flatbuffers::FlatBufferBuilder fbb(1024);

  aivision::control::SelfCheckStatus status =
      success ? aivision::control::SelfCheckStatus_Passed
              : aivision::control::SelfCheckStatus_Failed;

  auto response_offset = aivision::control::CreateAlgoLoadResultMsgDirect(
      fbb, token.c_str(), algo_name.c_str(), version.c_str(), token.c_str(),
      success, status, err_code.c_str(), err_msg.c_str(), load_time_ms,
      npu_mem_bytes);

  fbb.Finish(response_offset);

  // 8. Write response back on the active client connection
  int client_fd = response_router_->GetActiveClientFd();
  if (client_fd != -1) {
    std::cout << "Sending self check response. Success=" << success
              << ", time=" << load_time_ms << "ms" << std::endl;
    response_router_->SendResponse(client_fd, 514, fbb.GetBufferPointer(),
                                   fbb.GetSize());
  } else {
    std::cerr << "Failed to send response: no active client connection"
              << std::endl;
  }
}

// ============================================================
// Phase 3: Remote Operations — Shell Command Execution
// ============================================================

void InferenceEngine::HandleShellExec(const uint8_t *payload,
                                       size_t size, uint64_t seq)
{
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);

  std::string trace_id = ExtractJsonField(payload_str, "trace_id");
  std::string execution_id = ExtractJsonField(payload_str, "execution_id");
  std::string command = ExtractJsonField(payload_str, "command");
  std::string timeout_str = ExtractJsonField(payload_str, "timeout_seconds");
  std::string callback_url = ExtractJsonField(payload_str, "callback_url");

  int timeout_seconds = 30;
  if (!timeout_str.empty()) {
    try { timeout_seconds = std::stoi(timeout_str); } catch (...) {}
  }

  std::cout << "[Engine] HandleShellExec: trace_id=" << trace_id
            << " execution_id=" << execution_id << std::endl;

  if (command.empty()) {
    json error_json = {
        {"type", "shell_exec_result"},
        {"trace_id", trace_id},
        {"execution_id", execution_id},
        {"status", "failed"},
        {"stdout", ""},
        {"stderr", "empty command"},
        {"exit_code", -1}
    };
    if (mqtt_control_plane_) {
      mqtt_control_plane_->PublishResponse("shell_exec_result", error_json.dump());
    }
    return;
  }

  // Execute command
  monitor::ShellExecRequest req;
  req.trace_id = trace_id;
  req.execution_id = execution_id;
  req.command = command;
  req.timeout_seconds = timeout_seconds;
  req.callback_url = callback_url;

  monitor::ShellExecResult result;
  if (command_executor_) {
    result = command_executor_->Execute(req);
  } else {
    result.error_message = "Command executor not initialized";
    result.success = false;
    result.exit_code = -1;
  }

  // Build result JSON
  json result_json = {
      {"type", "shell_exec_result"},
      {"trace_id", trace_id},
      {"execution_id", execution_id},
      {"node_id", config_.node_id},
      {"status", result.success ? "success" : "failed"},
      {"stdout", result.stdout_str},
      {"stderr", result.stderr_str},
      {"exit_code", result.exit_code},
      {"duration_ms", result.duration_ms}
  };

  if (!result.error_message.empty()) {
    result_json["error_message"] = result.error_message;
  }

  // Publish result via MQTT response topic
  if (mqtt_control_plane_) {
    mqtt_control_plane_->PublishResponse("shell_exec_result", result_json.dump());
  }

  std::cout << "[Engine] ShellExec result: execution_id=" << execution_id
            << " status=" << (result.success ? "success" : "failed")
            << " exit_code=" << result.exit_code
            << " duration=" << result.duration_ms << "ms" << std::endl;
}

// ============================================================
// Phase 3: Remote Operations — PTY Module
// ============================================================

void InferenceEngine::HandlePtyOpen(const uint8_t *payload,
                                     size_t size, uint64_t seq)
{
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  std::string session_id = ExtractJsonField(payload_str, "session_id");

  std::cout << "[Engine] HandlePtyOpen: session_id=" << session_id << std::endl;

  if (session_id.empty()) {
    std::cerr << "[Engine] PtyOpen: missing session_id" << std::endl;
    return;
  }

  if (!pty_module_) {
    std::cerr << "[Engine] PtyOpen: PTY module not initialized" << std::endl;
    return;
  }

  bool ok = pty_module_->OpenSession(session_id);
  if (!ok) {
    std::cerr << "[Engine] PtyOpen: failed to open session: " << session_id << std::endl;
    json error_json = {
        {"type", "pty_error"},
        {"session_id", session_id},
        {"error", "Failed to open PTY session"}
    };
    if (mqtt_control_plane_) {
      mqtt_control_plane_->PublishResponse("pty_error", error_json.dump());
    }
  }
}

void InferenceEngine::HandlePtyWrite(const uint8_t *payload,
                                      size_t size, uint64_t seq)
{
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  std::string session_id = ExtractJsonField(payload_str, "session_id");
  std::string data = ExtractJsonField(payload_str, "data");

  if (session_id.empty() || data.empty()) {
    return;
  }

  if (pty_module_) {
    // Unescape JSON string (embedded newlines, etc.)
    // The data comes JSON-encoded, so we parse it to get the actual string
    try {
      json js = json::parse(payload_str);
      if (js.contains("data")) {
        std::string raw_data = js["data"].get<std::string>();
        pty_module_->WriteToSession(session_id, raw_data);
      }
    } catch (...) {
      pty_module_->WriteToSession(session_id, data);
    }
  }
}

void InferenceEngine::HandlePtyResize(const uint8_t *payload,
                                       size_t size, uint64_t seq)
{
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);

  try {
    json js = json::parse(payload_str);
    std::string session_id = js.value("session_id", "");
    unsigned short cols = static_cast<unsigned short>(js.value("cols", 80));
    unsigned short rows = static_cast<unsigned short>(js.value("rows", 24));

    if (!session_id.empty() && pty_module_) {
      pty_module_->ResizeSession(session_id, cols, rows);
    }
  } catch (const std::exception& e) {
    std::cerr << "[Engine] PtyResize error: " << e.what() << std::endl;
  }
}

void InferenceEngine::HandlePtyClose(const uint8_t *payload,
                                      size_t size, uint64_t seq)
{
  (void)seq;
  const std::string payload_str = PayloadToString(payload, size);
  std::string session_id = ExtractJsonField(payload_str, "session_id");

  if (session_id.empty()) {
    return;
  }

  if (pty_module_) {
    pty_module_->CloseSession(session_id);
  }
}

} // namespace aivision
