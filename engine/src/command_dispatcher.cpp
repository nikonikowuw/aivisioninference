#include "command_dispatcher.h"
#include "engine.h"
#include <iostream>
#include <nlohmann/json.hpp>
#include "proto/flatbuf/commands_generated.h"
#include "ipc/ipc_server.h"

using json = nlohmann::json;

namespace aivision
{
    CommandDispatcher::CommandDispatcher(InferenceEngine *engine)
        : engine_(engine) {}

    void CommandDispatcher::DispatchIPCCommand(uint32_t cmd_type, const uint8_t *payload, size_t size, int client_fd)
    {
        // client_fd is set via ScopedMqttContext for MQTT commands, or passed from IPC directly
        
        switch (cmd_type)
        {
        case 101: // HandleStartStream
            engine_->HandleStartStream(payload, size, 0);
            break;
        case 102: // HandleStopStream
            engine_->HandleStopStream(payload, size, 0);
            break;
        case 103: // HandleUpdateAlgoConfig
            engine_->HandleUpdateAlgoConfig(payload, size, 0);
            break;
        case 104: // HandleHeartbeat
            engine_->HandleHeartbeat(payload, size, 0);
            break;
        case 105: // HandleShutdown
            engine_->HandleShutdown(payload, size, 0);
            break;
        case 201: // StreamStart
            engine_->HandleStreamStart(payload, size, 0);
            break;
        case 202: // StreamStop
            engine_->HandleStreamStop(payload, size, 0);
            break;
        case 203: // StreamPlaybackStart
            engine_->HandleStreamPlaybackStart(payload, size, 0);
            break;
        case 204: // StreamPlaybackStop
            engine_->HandleStreamPlaybackStop(payload, size, 0);
            break;
        case 205: // StreamStatus
            engine_->HandleStreamStatus(payload, size, 0);
            break;
        case 206: // StartSelfCheck
            engine_->HandleStartSelfCheck(payload, size, 0);
            break;
        case 207: // FaceLibraryUpdate
            engine_->HandleFaceLibraryUpdate(payload, size, 0);
            break;
        case 208: // FaceEmbeddingExtract
            engine_->HandleFaceEmbeddingExtract(payload, size, 0);
            break;
        default:
            std::cerr << "[CommandDispatcher] Unknown IPC command type: " << cmd_type << std::endl;
            break;
        }
    }

    void CommandDispatcher::DispatchMqttCommand(const std::string &cmd_name, const std::string &payload_json)
    {
        json js;
        try
        {
            js = json::parse(payload_json);
        }
        catch (const std::exception &e)
        {
            std::cerr << "[CommandDispatcher] Failed to parse MQTT JSON: " << e.what() << std::endl;
            return;
        }

        std::string trace_id = js.value("trace_id", "");

        // 使用 RAII 守护类设置 thread_local 状态为 MQTT 响应模式
        ScopedMqttContext mqtt_guard(trace_id);

        std::cout << "[CommandDispatcher] Dispatching MQTT command: " << cmd_name 
                  << " trace_id=" << trace_id << std::endl;

        const uint8_t *payload_ptr = reinterpret_cast<const uint8_t *>(payload_json.data());
        size_t payload_size = payload_json.size();

        if (cmd_name == "start_stream")
        {
            engine_->HandleStreamStart(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "stop_stream")
        {
            engine_->HandleStreamStop(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "start_playback")
        {
            engine_->HandleStreamPlaybackStart(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "stop_playback")
        {
            engine_->HandleStreamPlaybackStop(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "stream_status")
        {
            engine_->HandleStreamStatus(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "self_check" || cmd_name == "start_self_check")
        {
            // SelfCheck expects a FlatBuffer payload
            std::string download_url = js.value("download_url", "");
            std::string token = js.value("token", "");
            std::string algo_name = js.value("algo_name", "");
            std::string version = js.value("version", "");

            flatbuffers::FlatBufferBuilder fbb(1024);
            auto url_offset = fbb.CreateString(download_url);
            auto token_offset = fbb.CreateString(token);
            auto name_offset = fbb.CreateString(algo_name);
            auto ver_offset = fbb.CreateString(version);

            auto cmd = aivision::ipc::CreateStartSelfCheckCmd(fbb, url_offset, token_offset, name_offset, ver_offset);
            fbb.Finish(cmd);

            engine_->HandleStartSelfCheck(fbb.GetBufferPointer(), fbb.GetSize(), 0);
        }
        else if (cmd_name == "face_library" || cmd_name == "face_library_update")
        {
            engine_->HandleFaceLibraryUpdate(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "face_embedding" || cmd_name == "face_embedding_extract")
        {
            if (js.contains("image_bytes") && !js.contains("image_base64")) {
                js["image_base64"] = js["image_bytes"];
                std::string adapted_payload = js.dump();
                engine_->HandleFaceEmbeddingExtract(
                    reinterpret_cast<const uint8_t*>(adapted_payload.data()), 
                    adapted_payload.size(), 
                    0);
            } else {
                engine_->HandleFaceEmbeddingExtract(payload_ptr, payload_size, 0);
            }
        }
        else
        {
            std::cerr << "[CommandDispatcher] Unknown MQTT command: " << cmd_name << std::endl;
        }
    }

} // namespace aivision
