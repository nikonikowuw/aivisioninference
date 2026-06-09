#ifndef AIVISION_PIPELINE_FFMPEG_FALLBACK_DECODER_H
#define AIVISION_PIPELINE_FFMPEG_FALLBACK_DECODER_H

#include <atomic>
#include <functional>
#include <memory>
#include <string>
#include <thread>

#include "pipeline/ring_queue.h"

namespace aivision
{
    namespace pipeline
    {

        class FFmpegFallbackDecoder
        {
        public:
            using FrameCallback = std::function<void(HwBufferPtr)>;

            FFmpegFallbackDecoder() = default;
            ~FFmpegFallbackDecoder();

            bool Start(const std::string &rtsp_url,
                       const std::string &device_id,
                       uint32_t output_width,
                       uint32_t output_height,
                       FrameCallback callback);
            void Stop();
            bool IsRunning() const { return running_.load(); }

        private:
            void DecodeLoop(std::string rtsp_url,
                            std::string device_id,
                            uint32_t output_width,
                            uint32_t output_height,
                            FrameCallback callback);

            std::atomic<bool> running_{false};
            std::unique_ptr<std::thread> thread_;
        };

    } // namespace pipeline
} // namespace aivision

#endif // AIVISION_PIPELINE_FFMPEG_FALLBACK_DECODER_H
