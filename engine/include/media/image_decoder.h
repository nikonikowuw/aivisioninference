/**
 * @file image_decoder.h
 * @brief 使用 FFmpeg 将受支持的压缩图片解码为算法 ABI 使用的 BGR24 数据。
 *
 * 解码器只接受人员图片上传契约允许的 JPEG、PNG 和 WebP，并在分配输出
 * 缓冲区前限制解码尺寸，避免压缩图片造成不受控的内存占用。
 */

#ifndef AIVISION_MEDIA_IMAGE_DECODER_H
#define AIVISION_MEDIA_IMAGE_DECODER_H

#include <cstddef>
#include <cstdint>
#include <vector>

namespace aivision::media {

/** DecodeImageResult 保存连续 BGR24 图片及其内存布局。 */
struct DecodeImageResult {
  std::vector<uint8_t> pixels;
  uint32_t width = 0;
  uint32_t height = 0;
  uint32_t stride = 0;
};

/**
 * @brief DecodeImageToBGR24 解码 JPEG、PNG 或 WebP 图片并转换为 BGR24。
 * @param encoded 压缩图片数据。
 * @param encoded_size 压缩图片字节数，必须能由 int 表示。
 * @param result 接收连续 BGR24 数据；失败时被重置为空结果。
 * @return 解码及像素格式转换成功时返回 true，否则返回 false。
 */
bool DecodeImageToBGR24(const uint8_t *encoded, size_t encoded_size,
                        DecodeImageResult *result);

} // namespace aivision::media

#endif // AIVISION_MEDIA_IMAGE_DECODER_H
