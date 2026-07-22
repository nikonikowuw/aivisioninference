// 自定义 JSON 格式化器，用于文件 sink。
//
// 每行一个 JSON 对象（JSON Lines 格式）:
//   {"ts":"2026-07-22T14:30:01.123456Z","lvl":"INFO","file":"main.cpp","line":190,"tid":12345,"pid":67890,"msg":"..."}
//
// 时间戳使用 UTC ISO 8601，微秒精度。

#ifndef AIVISION_JSON_FORMATTER_H
#define AIVISION_JSON_FORMATTER_H

#include <spdlog/formatter.h>

namespace aivision {

class JsonFormatter : public spdlog::formatter {
public:
    void format(const spdlog::details::log_msg &msg, spdlog::memory_buf_t &dest) override;
    std::unique_ptr<formatter> clone() const override;

private:
    /// 将字符串以 JSON 转义方式追加到 dest。
    static void AppendJsonString(spdlog::memory_buf_t &dest, const char *str, size_t len);
};

} // namespace aivision

#endif // AIVISION_JSON_FORMATTER_H
