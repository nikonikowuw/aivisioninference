#ifndef AIVISION_HTTP_SERVER_H
#define AIVISION_HTTP_SERVER_H

#include <memory>
#include <thread>
#include <atomic>
#include <string>

namespace aivision
{
    class InferenceEngine;

    namespace http
    {
        class HTTPServer
        {
        public:
            explicit HTTPServer(InferenceEngine* engine);
            ~HTTPServer();

            bool Start(int port);
            void Stop();

        private:
            void RunServer(int port);
            
            InferenceEngine* engine_;
            std::unique_ptr<std::thread> thread_;
            std::atomic<void*> svr_ptr_{nullptr};
            std::atomic<bool> running_{false};
            int port_{-1};
        };
    }
}

#endif // AIVISION_HTTP_SERVER_H
