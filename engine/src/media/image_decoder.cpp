/**
 * @file image_decoder.cpp
 * @brief 基于 FFmpeg codec 与 libswscale 实现单张图片安全解码。
 */

#include "media/image_decoder.h"

#include <algorithm>
#include <climits>
#include <cstring>
#include <limits>
#include <memory>
#include <utility>

extern "C" {
#include <libavcodec/avcodec.h>
#include <libavutil/frame.h>
#include <libavutil/pixfmt.h>
#include <libswscale/swscale.h>
}

namespace aivision::media {
namespace {

constexpr uint32_t kMaxImageDimension = 16384;
constexpr size_t kMaxDecodedImageBytes = 256ULL * 1024ULL * 1024ULL;
constexpr int64_t kMaxImagePixels =
    static_cast<int64_t>(kMaxDecodedImageBytes / 3);

struct CodecContextDeleter {
  void operator()(AVCodecContext *context) const {
    avcodec_free_context(&context);
  }
};

struct PacketDeleter {
  void operator()(AVPacket *packet) const { av_packet_free(&packet); }
};

struct FrameDeleter {
  void operator()(AVFrame *frame) const { av_frame_free(&frame); }
};

AVCodecID DetectCodec(const uint8_t *data, size_t size) {
  static constexpr uint8_t kPngSignature[] = {0x89, 0x50, 0x4e, 0x47,
                                              0x0d, 0x0a, 0x1a, 0x0a};
  if (size >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff) {
    return AV_CODEC_ID_MJPEG;
  }
  if (size >= sizeof(kPngSignature) &&
      std::memcmp(data, kPngSignature, sizeof(kPngSignature)) == 0) {
    return AV_CODEC_ID_PNG;
  }
  if (size >= 12 && std::memcmp(data, "RIFF", 4) == 0 &&
      std::memcmp(data + 8, "WEBP", 4) == 0) {
    return AV_CODEC_ID_WEBP;
  }
  return AV_CODEC_ID_NONE;
}

bool IsSafeOutputSize(int width, int height, size_t *output_size) {
  if (!output_size || width <= 0 || height <= 0 ||
      width > static_cast<int>(kMaxImageDimension) ||
      height > static_cast<int>(kMaxImageDimension)) {
    return false;
  }
  const size_t row_bytes = static_cast<size_t>(width) * 3;
  if (static_cast<size_t>(height) >
      std::numeric_limits<size_t>::max() / row_bytes) {
    return false;
  }
  *output_size = row_bytes * static_cast<size_t>(height);
  return *output_size <= kMaxDecodedImageBytes;
}

} // namespace

bool DecodeImageToBGR24(const uint8_t *encoded, size_t encoded_size,
                        DecodeImageResult *result) {
  if (!result) {
    return false;
  }
  *result = {};
  if (!encoded || encoded_size == 0 || encoded_size > INT_MAX) {
    return false;
  }

  const AVCodecID codec_id = DetectCodec(encoded, encoded_size);
  const AVCodec *codec = avcodec_find_decoder(codec_id);
  if (!codec) {
    return false;
  }

  using CodecContextPtr = std::unique_ptr<AVCodecContext, CodecContextDeleter>;
  CodecContextPtr codec_context(avcodec_alloc_context3(codec));
  if (!codec_context) {
    return false;
  }
  // 在 codec 分配帧缓冲区前限制像素数量，防止压缩图片触发超大分配。
  codec_context->max_pixels = kMaxImagePixels;
  if (avcodec_open2(codec_context.get(), codec, nullptr) < 0) {
    return false;
  }

  using PacketPtr = std::unique_ptr<AVPacket, PacketDeleter>;
  PacketPtr packet(av_packet_alloc());
  if (!packet ||
      av_new_packet(packet.get(), static_cast<int>(encoded_size)) < 0) {
    return false;
  }
  std::memcpy(packet->data, encoded, encoded_size);

  using FramePtr = std::unique_ptr<AVFrame, FrameDeleter>;
  FramePtr frame(av_frame_alloc());
  if (!frame || avcodec_send_packet(codec_context.get(), packet.get()) < 0 ||
      avcodec_receive_frame(codec_context.get(), frame.get()) < 0) {
    return false;
  }

  size_t output_size = 0;
  if (!IsSafeOutputSize(frame->width, frame->height, &output_size)) {
    return false;
  }

  using SwsContextPtr = std::unique_ptr<SwsContext, decltype(&sws_freeContext)>;
  SwsContextPtr sws_context(
      sws_getContext(frame->width, frame->height,
                     static_cast<AVPixelFormat>(frame->format), frame->width,
                     frame->height, AV_PIX_FMT_BGR24, SWS_BILINEAR, nullptr,
                     nullptr, nullptr),
      &sws_freeContext);
  if (!sws_context) {
    return false;
  }

  DecodeImageResult decoded;
  decoded.width = static_cast<uint32_t>(frame->width);
  decoded.height = static_cast<uint32_t>(frame->height);
  decoded.stride = decoded.width * 3;
  decoded.pixels.resize(output_size);
  uint8_t *destination_data[4] = {decoded.pixels.data(), nullptr, nullptr,
                                  nullptr};
  int destination_linesize[4] = {static_cast<int>(decoded.stride), 0, 0, 0};
  const int converted_rows =
      sws_scale(sws_context.get(), frame->data, frame->linesize, 0,
                frame->height, destination_data, destination_linesize);
  if (converted_rows != frame->height) {
    return false;
  }

  *result = std::move(decoded);
  return true;
}

} // namespace aivision::media
