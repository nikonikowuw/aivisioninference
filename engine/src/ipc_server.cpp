
#include "ipc/ipc_server.h"
#include <iostream>
#include <string>

namespace aivision
{
    ScopedMqttContext::ScopedMqttContext(const std::string &trace_id)
    {
        ipc::ResponseRouter::SetActiveClientFd(-2);
        ipc::ResponseRouter::SetActiveMqttTraceId(trace_id);
    }

    ScopedMqttContext::~ScopedMqttContext()
    {
        ipc::ResponseRouter::SetActiveClientFd(-1);
        ipc::ResponseRouter::SetActiveMqttTraceId("");
    }

    namespace ipc
    {
        static thread_local int t_active_client_fd = -1;
        static thread_local std::string t_active_mqtt_trace_id = "";

        ResponseRouter::ResponseRouter() {}
        ResponseRouter::~ResponseRouter() = default;

        int ResponseRouter::GetActiveClientFd()
        {
            return t_active_client_fd;
        }

        void ResponseRouter::SetActiveClientFd(int fd)
        {
            t_active_client_fd = fd;
        }

        void ResponseRouter::SetActiveMqttTraceId(const std::string &trace_id)
        {
            t_active_mqtt_trace_id = trace_id;
        }

        std::string ResponseRouter::GetActiveMqttTraceId()
        {
            return t_active_mqtt_trace_id;
        }

        bool ResponseRouter::SendResponse(int client_fd, uint32_t resp_type, const uint8_t *payload, size_t payload_len)
        {
            if (client_fd == -2 && mqtt_response_callback_)
            {
                mqtt_response_callback_(resp_type, payload, payload_len);
                return true;
            }
            return false;
        }
    }
} // namespace aivision
