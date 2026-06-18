#pragma once

#include <filesystem>
#include <string>

namespace aivision {
namespace algo {

constexpr const char *kAlgorithmSoName = "nikoniko_detector.so";

inline std::string FindAlgorithmSo(const std::string &package_dir) {
  namespace fs = std::filesystem;
  if (package_dir.empty()) {
    return "";
  }

  std::error_code ec;
  const fs::path root = fs::weakly_canonical(fs::path(package_dir), ec);
  if (ec || !fs::exists(root, ec) || !fs::is_directory(root, ec)) {
    return "";
  }

  const fs::path detector_so = root / kAlgorithmSoName;
  if (fs::exists(detector_so, ec) && fs::is_regular_file(detector_so, ec)) {
    return detector_so.string();
  }

  return "";
}

} // namespace algo
} // namespace aivision
