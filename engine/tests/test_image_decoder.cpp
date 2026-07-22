/**
 * @file test_image_decoder.cpp
 * @brief 验证 FFmpeg 图片解码器覆盖人员图片契约允许的常用格式。
 */

#include "media/image_decoder.h"

#include <algorithm>
#include <cstdint>
#include <iostream>
#include <iterator>
#include <string>
#include <vector>

namespace {

int passed = 0;
int failed = 0;

#define TEST(name, expression)                                                 \
  do {                                                                         \
    if (expression) {                                                          \
      std::cout << "[PASS] " << name << '\n';                                  \
      ++passed;                                                                \
    } else {                                                                   \
      std::cerr << "[FAIL] " << name << '\n';                                  \
      ++failed;                                                                \
    }                                                                          \
  } while (false)

std::vector<uint8_t> DecodeBase64(const std::string &input) {
  static constexpr char kAlphabet[] =
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  std::vector<uint8_t> output;
  uint32_t accumulator = 0;
  int bits = -8;
  for (const char character : input) {
    if (character == '=') {
      break;
    }
    const char *position =
        std::find(std::begin(kAlphabet), std::end(kAlphabet) - 1, character);
    if (position == std::end(kAlphabet) - 1) {
      continue;
    }
    accumulator =
        (accumulator << 6) + static_cast<uint32_t>(position - kAlphabet);
    bits += 6;
    if (bits >= 0) {
      output.push_back(static_cast<uint8_t>((accumulator >> bits) & 0xff));
      bits -= 8;
    }
  }
  return output;
}

void TestImage(const char *name, const std::string &base64,
               uint32_t expected_width, uint32_t expected_height) {
  const std::vector<uint8_t> encoded = DecodeBase64(base64);
  aivision::media::DecodeImageResult image;
  const bool decoded = aivision::media::DecodeImageToBGR24(
      encoded.data(), encoded.size(), &image);
  TEST(std::string(name) + " decodes", decoded);
  TEST(std::string(name) + " has expected dimensions",
       decoded && image.width == expected_width &&
           image.height == expected_height);
  TEST(std::string(name) + " is tightly packed BGR24",
       decoded && image.stride == expected_width * 3 &&
           image.pixels.size() ==
               static_cast<size_t>(expected_width) * expected_height * 3);
}

} // namespace

int main() {
  TestImage("JPEG",
            "/9j/4AAQSkZJRgABAgAAAQABAAD//gAQTGF2YzYyLjI4LjEwMgD/2wBDAAgEBAQE"
            "BAUFBQUFBQYGBgYGBgYGBgYGBgYHBwcICAgHBwcGBgcHCAgICAkJCQgICAgJCQoK"
            "CgwMCwsODg4RERT/xABMAAEBAAAAAAAAAAAAAAAAAAAABgEBAQAAAAAAAAAAAAAAAA"
            "AABgcQAQAAAAAAAAAAAAAAAAAAAAARAQAAAAAAAAAAAAAAAAAAAAD/wAARCAACAAID"
            "ASIAAhEAAxEA/9oADAMBAAIRAxEAPwCLAE1/f//Z",
            2, 2);
  TestImage(
      "PNG",
      "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAACXBIWXMAAAABAAAAAQBP"
      "JcTWAAAAEElEQVR4nGP8wwACLGCSAQANBAECv1AVswAAAABJRU5ErkJggg==",
      2, 2);
  TestImage("WebP", "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA",
            1, 1);

  const std::vector<uint8_t> invalid = {'n', 'o', 't', ' ', 'a', 'n',
                                        ' ', 'i', 'm', 'a', 'g', 'e'};
  aivision::media::DecodeImageResult image;
  TEST("invalid input is rejected",
       !aivision::media::DecodeImageToBGR24(invalid.data(), invalid.size(),
                                            &image));
  TEST("failed decode clears output",
       image.pixels.empty() && image.width == 0 && image.height == 0 &&
           image.stride == 0);

  std::cout << passed << " passed, " << failed << " failed\n";
  return failed == 0 ? 0 : 1;
}
