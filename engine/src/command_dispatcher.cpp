#include "command_dispatcher.h"
#include "engine.h"
#include "response_router.h"

#include <iostream>
#include <nlohmann/json.hpp>

#include "proto/flatbuf/commands_generated.h"

using json = nlohmann::json;

namespace aivision
{
    CommandDispatcher::CommandDispatcher(InferenceEngine *engine)
        : engine_(engine) {}

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

            auto cmd = aivision::control::CreateStartSelfCheckCmd(fbb, url_offset, token_offset, name_offset, ver_offset);
            fbb.Finish(cmd);

            engine_->HandleStartSelfCheck(fbb.GetBufferPointer(), fbb.GetSize(), 0);
        }
        else if (cmd_name == "face_library" || cmd_name == "face_library_update")
        {
            engine_->HandleFaceLibraryUpdate(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "algo_warmup" || cmd_name == "algorithm_warmup")
        {
            engine_->HandleAlgoWarmup(payload_ptr, payload_size, 0);
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
        else if (cmd_name == "shell_exec")
        {
            engine_->HandleShellExec(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "pty_open")
        {
            engine_->HandlePtyOpen(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "pty_write")
        {
            engine_->HandlePtyWrite(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "pty_resize")
        {
            engine_->HandlePtyResize(payload_ptr, payload_size, 0);
        }
        else if (cmd_name == "pty_close")
        {
            engine_->HandlePtyClose(payload_ptr, payload_size, 0);
        }
        else
        {
            std::cerr << "[CommandDispatcher] Unknown MQTT command: " << cmd_name << std::endl;
        }
    }

} // namespace aivision
