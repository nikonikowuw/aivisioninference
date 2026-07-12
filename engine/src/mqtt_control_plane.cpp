#include "mqtt_control_plane.h"
#include "engine.h"
#include "command_dispatcher.h"
#include "mqtt/async_client.h"
#include "proto/flatbuf/envelope_generated.h"
#include <iostream>
#include <chrono>
#include <thread>
#include <vector>
#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace aivision
{
    struct MqttControlPlane::Impl : public virtual mqtt::callback
    {
        MqttControlPlane *parent_;
        std::unique_ptr<mqtt::async_client> client_;
        bool connected_{false};
        std::string node_id_;

        Impl(MqttControlPlane *parent, const std::string &server_uri, const std::string &client_id, const std::string &node_id)
            : parent_(parent), node_id_(node_id)
        {
            client_ = std::make_unique<mqtt::async_client>(server_uri, client_id);
            client_->set_callback(*this);
        }

        ~Impl() override
        {
            try
            {
                if (client_ && client_->is_connected())
                {
                    client_->disconnect()->wait();
                }
            }
            catch (...) {}
        }

        void connection_lost(const std::string &cause) override
        {
            std::cerr << "[MQTT] Connection lost: " << cause << std::endl;
            connected_ = false;
        }

        void message_arrived(mqtt::const_message_ptr msg) override
        {
            std::string topic = msg->get_topic();
            std::string payload = msg->to_string();

            // Offload message dispatching to io_context pool to avoid blocking MQTT loop and resource exhaustion
            asio::post(parent_->io_context_, [this, topic, payload]() {
                try
                {
                    // Robust topic parsing: extract command name before "/cmd" if it exists, otherwise use last part
                    std::string cmd_name = "";
                    std::vector<std::string> parts;
                    size_t start = 0;
                    size_t end = topic.find('/');
                    while (end != std::string::npos)
                    {
                        parts.push_back(topic.substr(start, end - start));
                        start = end + 1;
                        end = topic.find('/', start);
                    }
                    parts.push_back(topic.substr(start));

                    if (parts.empty()) return;

                    if (parts.back() == "cmd" && parts.size() >= 2)
                    {
                        cmd_name = parts[parts.size() - 2];
                    }
                    else
                    {
                        cmd_name = parts.back();
                    }

                    if (cmd_name.empty()) return;

                    parent_->engine_->GetCommandDispatcher()->DispatchMqttCommand(cmd_name, payload);
                }
                catch (const std::exception &e)
                {
                    std::cerr << "[MQTT] Exception in message processing task: " << e.what() << std::endl;
                }
            });
        }

        void delivery_complete(mqtt::delivery_token_ptr tok) override {}
    };

    MqttControlPlane::MqttControlPlane(InferenceEngine *engine)
        : engine_(engine),
          work_guard_(std::make_unique<asio::executor_work_guard<asio::io_context::executor_type>>(io_context_.get_executor()))
    {
        // Start a small thread pool for MQTT command processing
        for (int i = 0; i < 2; ++i)
        {
            worker_threads_.emplace_back([this]() {
                io_context_.run();
            });
        }

        std::string broker = engine_->GetConfig().mqtt_broker;
        std::string node_id = engine_->GetConfig().node_id;
        if (node_id.empty())
        {
            node_id = "default-node";
        }
        std::string client_id = engine_->GetConfig().mqtt_client_id;
        if (client_id.empty())
        {
            client_id = "aivision-engine-" + node_id;
        }

        impl_ = std::make_unique<Impl>(this, broker, client_id, node_id);
    }

    MqttControlPlane::~MqttControlPlane()
    {
        Stop();
        work_guard_.reset();
        io_context_.stop();
        for (auto &t : worker_threads_)
        {
            if (t.joinable())
                t.join();
        }
    }

    bool MqttControlPlane::Start()
    {
        try
        {
            mqtt::connect_options connOpts;
            connOpts.set_keep_alive_interval(20);
            connOpts.set_clean_session(true);
            connOpts.set_automatic_reconnect(true);

            if (!engine_->GetConfig().mqtt_username.empty())
            {
                connOpts.set_user_name(engine_->GetConfig().mqtt_username);
                connOpts.set_password(engine_->GetConfig().mqtt_password);
            }

            // Setup Last Will and Testament (LWT)
            std::string lwt_topic = "aivision/edge/" + impl_->node_id_ + "/status/lifecycle";
            mqtt::will_options will(lwt_topic, std::string("offline"), 1, true);
            connOpts.set_will(will);

            std::cout << "[MQTT] Connecting to broker " << engine_->GetConfig().mqtt_broker << "..." << std::endl;
            impl_->client_->connect(connOpts)->wait();
            impl_->connected_ = true;
            std::cout << "[MQTT] Connected successfully." << std::endl;

            // Subscribe to control topics
            std::string cmd_topic = "aivision/edge/" + impl_->node_id_ + "/cmd/#";
            impl_->client_->subscribe(cmd_topic, 1)->wait();

            impl_->client_->subscribe("aivision/edge/self_check/cmd", 1)->wait();

            std::cout << "[MQTT] Subscribed to command topics." << std::endl;

            // Publish lifecycle "online"
            impl_->client_->publish(lwt_topic, "online", 1, true)->wait();

            return true;
        }
        catch (const mqtt::exception &exc)
        {
            std::cerr << "[MQTT] Connection failed: " << exc.what() << std::endl;
            return false;
        }
    }

    void MqttControlPlane::Stop()
    {
        try
        {
            if (impl_->client_ && impl_->client_->is_connected())
            {
                // Publish offline state before disconnecting
                std::string lwt_topic = "aivision/edge/" + impl_->node_id_ + "/status/lifecycle";
                impl_->client_->publish(lwt_topic, "offline", 1, true)->wait();
                impl_->client_->disconnect()->wait();
            }
        }
        catch (...) {}
        impl_->connected_ = false;
    }

    bool MqttControlPlane::PublishEvent(uint16_t signal_type, flatbuffers::FlatBufferBuilder &fbb)
    {
        if (!IsConnected()) return false;
        try
        {
            // Wrap in ControlEnvelope
            flatbuffers::FlatBufferBuilder env_fbb(fbb.GetSize() + 256);
            auto payload_offset = env_fbb.CreateVector(fbb.GetBufferPointer(), fbb.GetSize());

            aivision::control::ControlEnvelopeBuilder env_builder(env_fbb);
            env_builder.add_schema_version(102);
            env_builder.add_signal_type(static_cast<aivision::control::SignalType>(signal_type));
            env_builder.add_timestamp_ns(std::chrono::duration_cast<std::chrono::nanoseconds>(
                std::chrono::system_clock::now().time_since_epoch()).count());
            env_builder.add_payload(payload_offset);

            auto env_offset = env_builder.Finish();
            env_fbb.Finish(env_offset);

            std::string topic = "";
            if (signal_type == 0x0200) // InferenceResult
            {
                topic = "aivision/edge/" + impl_->node_id_ + "/event/inference";
            }
            else if (signal_type == 0x0203) // EngineMetrics
            {
                topic = "aivision/edge/" + impl_->node_id_ + "/event/metrics";
            }
            else
            {
                topic = "aivision/edge/" + impl_->node_id_ + "/event/generic";
            }

            impl_->client_->publish(topic, env_fbb.GetBufferPointer(), env_fbb.GetSize(), 0, false);
            return true;
        }
        catch (const mqtt::exception &exc)
        {
            std::cerr << "[MQTT] Publish event failed: " << exc.what() << std::endl;
            return false;
        }
    }

    bool MqttControlPlane::PublishResponse(const std::string &cmd_type, const std::string &payload_json)
    {
        if (!IsConnected()) return false;
        try
        {
            std::string topic = "aivision/edge/" + impl_->node_id_ + "/response/" + cmd_type;
            impl_->client_->publish(topic, payload_json, 1, false);
            return true;
        }
        catch (const mqtt::exception &exc)
        {
            std::cerr << "[MQTT] Publish response failed: " << exc.what() << std::endl;
            return false;
        }
    }

    bool MqttControlPlane::IsConnected() const
    {
        return impl_->connected_ && impl_->client_ && impl_->client_->is_connected();
    }

} // namespace aivision
