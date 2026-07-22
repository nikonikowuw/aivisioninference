#ifndef AIVISION_STRING_UTILS_H
#define AIVISION_STRING_UTILS_H

#include <string>
#include <string_view>

namespace aivision {

/// 去除字符串首尾空白字符（空格、制表符、回车、换行）
inline std::string Trim(std::string_view s) {
    auto a = s.find_first_not_of(" \t\r\n");
    if (a == std::string_view::npos) return {};
    auto b = s.find_last_not_of(" \t\r\n");
    return std::string(s.substr(a, b - a + 1));
}

}  // namespace aivision

#endif  // AIVISION_STRING_UTILS_H
