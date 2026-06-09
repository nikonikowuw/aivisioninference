#include "pipeline/ffmpeg_fallback_decoder.h"

#include <chrono>
#include <cstring>
#include <iostream>
#include <vector>

#ifdef AIVISION_WITH_FFMPEG_OPENCV
extern "C" {
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
}
#include <opencv2/imgproc.hpp>
#endif

namespace aivision
{
    namespace pipeline
    {
        namespace
        {
            constexpr uint32_t PixelFormatBGR24 = ('B') | ('G' << 8) | ('R' << 16) | ('3' << 24);
        }

        FFmpegFallbackDecoder::~FFmpegFallbackDecoder()
        {
            Stop();
        }

        bool FFmpegFallbackDecoder::Start(const std::string &rtsp_url,
                                          const std::string &device_id,
                                          uint32_t output_width,
                                          uint32_t output_height,
                                          FrameCallback callback)
        {
#ifndef AIVISION_WITH_FFMPEG_OPENCV
            (void)rtsp_url;
            (void)device_id;
            (void)output_width;
            (void)output_height;
            (void)callback;
            std::cerr << "[FFmpegFallback] software decode fallback is not compiled in; install libavformat/libavcodec/libavutil/libswscale and opencv4 dev packages" << std::endl;
            return false;
#else
            if (running_.load())
            {
                return true;
            }
            if (!callback)
            {
                std::cerr << "[FFmpegFallback] frame callback is empty" << std::endl;
                return false;
            }
            running_.store(true);
            thread_ = std::make_unique<std::thread>(&FFmpegFallbackDecoder::DecodeLoop,
                                                    this,
                                                    rtsp_url,
                                                    device_id,
                                                    output_width,
                                                    output_height,
                                                    std::move(callback));
            return true;
#endif
        }

        void FFmpegFallbackDecoder::Stop()
        {
            running_.store(false);
            if (thread_ && thread_->joinable())
            {
                thread_->join();
            }
            thread_.reset();
        }

        void FFmpegFallbackDecoder::DecodeLoop(std::string rtsp_url,
                                               std::string device_id,
                                               uint32_t output_width,
                                               uint32_t output_height,
                                               FrameCallback callback)
        {
#ifndef AIVISION_WITH_FFMPEG_OPENCV
            (void)rtsp_url;
            (void)device_id;
            (void)output_width;
            (void)output_height;
            (void)callback;
#else
            AVFormatContext *fmt_ctx = nullptr;
            AVDictionary *opts = nullptr;
            av_dict_set(&opts, "rtsp_transport", "tcp", 0);
            av_dict_set(&opts, "stimeout", "5000000", 0);

            if (avformat_open_input(&fmt_ctx, rtsp_url.c_str(), nullptr, &opts) < 0)
            {
                std::cerr << "[FFmpegFallback] failed to open input device=" << device_id
                          << ", url=" << rtsp_url << std::endl;
                av_dict_free(&opts);
                running_.store(false);
                return;
            }
            av_dict_free(&opts);

            if (avformat_find_stream_info(fmt_ctx, nullptr) < 0)
            {
                std::cerr << "[FFmpegFallback] failed to read stream info device=" << device_id << std::endl;
                avformat_close_input(&fmt_ctx);
                running_.store(false);
                return;
            }

            int video_index = av_find_best_stream(fmt_ctx, AVMEDIA_TYPE_VIDEO, -1, -1, nullptr, 0);
            if (video_index < 0)
            {
                std::cerr << "[FFmpegFallback] no video stream device=" << device_id << std::endl;
                avformat_close_input(&fmt_ctx);
                running_.store(false);
                return;
            }

            AVStream *stream = fmt_ctx->streams[video_index];
            const AVCodec *codec = avcodec_find_decoder(stream->codecpar->codec_id);
            if (!codec)
            {
                std::cerr << "[FFmpegFallback] decoder not found device=" << device_id << std::endl;
                avformat_close_input(&fmt_ctx);
                running_.store(false);
                return;
            }

            AVCodecContext *codec_ctx = avcodec_alloc_context3(codec);
            avcodec_parameters_to_context(codec_ctx, stream->codecpar);
            if (avcodec_open2(codec_ctx, codec, nullptr) < 0)
            {
                std::cerr << "[FFmpegFallback] failed to open decoder device=" << device_id << std::endl;
                avcodec_free_context(&codec_ctx);
                avformat_close_input(&fmt_ctx);
                running_.store(false);
                return;
            }

            uint32_t target_width = output_width > 0 ? output_width : static_cast<uint32_t>(codec_ctx->width);
            uint32_t target_height = output_height > 0 ? output_height : static_cast<uint32_t>(codec_ctx->height);
            AVFrame *frame = av_frame_alloc();
            AVFrame *bgr_frame = av_frame_alloc();
            AVPacket *packet = av_packet_alloc();
            int bgr_size = av_image_get_buffer_size(AV_PIX_FMT_BGR24, target_width, target_height, 1);
            std::vector<uint8_t> bgr_buffer(static_cast<size_t>(bgr_size));
            av_image_fill_arrays(bgr_frame->data, bgr_frame->linesize, bgr_buffer.data(), AV_PIX_FMT_BGR24,
                                 target_width, target_height, 1);

            SwsContext *sws = sws_getContext(codec_ctx->width, codec_ctx->height, codec_ctx->pix_fmt,
                                             target_width, target_height, AV_PIX_FMT_BGR24,
                                             SWS_BILINEAR, nullptr, nullptr, nullptr);
            uint64_t frame_count = 0;
            std::cout << "[FFmpegFallback] decoder started device=" << device_id
                      << " output=" << target_width << "x" << target_height << std::endl;

            while (running_.load() && av_read_frame(fmt_ctx, packet) >= 0)
            {
                if (packet->stream_index != video_index)
                {
                    av_packet_unref(packet);
                    continue;
                }
                if (avcodec_send_packet(codec_ctx, packet) == 0)
                {
                    while (running_.load() && avcodec_receive_frame(codec_ctx, frame) == 0)
                    {
                        sws_scale(sws, frame->data, frame->linesize, 0, codec_ctx->height,
                                  bgr_frame->data, bgr_frame->linesize);

                        auto frame_data = std::make_shared<std::vector<uint8_t>>(bgr_buffer.begin(), bgr_buffer.end());
                        HwBufferDesc desc;
                        desc.memory_type = HwBufferMemoryType::HostMemory;
                        desc.dma_fd = -1;
                        desc.data = frame_data->data();
                        desc.stride = static_cast<uint32_t>(bgr_frame->linesize[0]);
                        desc.size = frame_data->size();
                        desc.width = target_width;
                        desc.height = target_height;
                        desc.pixel_format = PixelFormatBGR24;
                        desc.native_handle = frame_data.get();
                        auto buffer = MakeHwBuffer(desc);
                        buffer->SetReleaseCallback([frame_data](const HwBufferDesc &) {});
                        callback(buffer);

                        frame_count++;
                        if (frame_count == 1 || frame_count % 100 == 0)
                        {
                            std::cout << "[FFmpegFallback] frame decoded device=" << device_id
                                      << " count=" << frame_count << std::endl;
                        }
                    }
                }
                av_packet_unref(packet);
            }

            std::cout << "[FFmpegFallback] decoder stopped device=" << device_id << std::endl;
            sws_freeContext(sws);
            av_packet_free(&packet);
            av_frame_free(&bgr_frame);
            av_frame_free(&frame);
            avcodec_free_context(&codec_ctx);
            avformat_close_input(&fmt_ctx);
            running_.store(false);
#endif
        }

    } // namespace pipeline
} // namespace aivision
