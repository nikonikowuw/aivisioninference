// JSON Lines 格式化器实现。
// 手动 JSON 转义避免额外依赖。使用 spdlog::memory_buf_t (fmt::buffer) API。

#include "logger/json_formatter.h"

#include <ctime>
#include <cstring>
#include <sys/time.h>
#include <unistd.h>

namespace aivision {

void JsonFormatter::format(const spdlog::details::log_msg &msg,
                           spdlog::memory_buf_t &dest)
{
    // --- 时间戳: UTC ISO 8601 微秒 ---
    struct timeval tv = {};
    struct tm tm_buf = {};
    (void)::gettimeofday(&tv, nullptr);
    ::gmtime_r(&tv.tv_sec, &tm_buf);

    char ts_buf[64];
    auto n = static_cast<size_t>(::strftime(ts_buf, sizeof(ts_buf),
                                            "%Y-%m-%dT%H:%M:%S", &tm_buf));
    int us = static_cast<int>(tv.tv_usec);
    char us_buf[8];
    auto us_len = static_cast<size_t>(::snprintf(us_buf, sizeof(us_buf), ".%06d", us));

    // --- 级别 ---
    const char *lvl = "TRACE";
    switch (msg.level) {
    case spdlog::level::trace:    lvl = "TRACE"; break;
    case spdlog::level::debug:    lvl = "DEBUG"; break;
    case spdlog::level::info:     lvl = "INFO";  break;
    case spdlog::level::warn:     lvl = "WARN";  break;
    case spdlog::level::err:      lvl = "ERROR"; break;
    case spdlog::level::critical: lvl = "FATAL"; break;
    default: break;
    }

    // --- 源文件名 (仅 basename，不含路径) ---
    const char *file = msg.source.filename ? msg.source.filename : "";
    const char *slash = nullptr;
    if (file && file[0]) {
        for (const char *p = file; *p; ++p) {
            if (*p == '/' || *p == '\\') slash = p;
        }
        if (slash) file = slash + 1;
    }
    int line = msg.source.line;

    // --- 进程 ID (缓存) ---
    static const pid_t pid = ::getpid();

    // --- 组装 JSON ---
    // 使用 fmt::buffer::append(ptr, ptr+len) 接口
    auto append_str = [&dest](const char *s, size_t len) {
        dest.append(s, s + len);
    };
    auto append_cstr = [&append_str](const char *s) {
        append_str(s, ::strlen(s));
    };

    append_cstr("{\"ts\":\"");
    append_str(ts_buf, n);
    append_str(us_buf, us_len);
    dest.push_back('Z');
    append_cstr("\",\"lvl\":\"");
    append_cstr(lvl);
    append_cstr("\",\"file\":\"");
    AppendJsonString(dest, file, ::strlen(file));
    append_cstr("\",\"line\":");
    // 行号
    char line_buf[16];
    auto line_n = static_cast<size_t>(::snprintf(line_buf, sizeof(line_buf), "%d", line));
    append_str(line_buf, line_n);
    append_cstr(",\"tid\":");
    // 线程 ID
    char tid_buf[32];
    auto tid_n = static_cast<size_t>(::snprintf(tid_buf, sizeof(tid_buf), "%zu",
                                                msg.thread_id));
    append_str(tid_buf, tid_n);
    append_cstr(",\"pid\":");
    char pid_buf[16];
    auto pid_n = static_cast<size_t>(::snprintf(pid_buf, sizeof(pid_buf), "%d",
                                                static_cast<int>(pid)));
    append_str(pid_buf, pid_n);
    append_cstr(",\"msg\":\"");
    AppendJsonString(dest, msg.payload.data(), msg.payload.size());
    append_cstr("\"}\n");
}

std::unique_ptr<spdlog::formatter> JsonFormatter::clone() const
{
    return std::make_unique<JsonFormatter>();
}

void JsonFormatter::AppendJsonString(spdlog::memory_buf_t &dest,
                                     const char *str, size_t len)
{
    auto append_str = [&dest](const char *s, size_t n) {
        dest.append(s, s + n);
    };

    for (size_t i = 0; i < len; ++i) {
        unsigned char c = static_cast<unsigned char>(str[i]);
        switch (c) {
        case '"':  append_str("\\\"", 2); break;
        case '\\': append_str("\\\\", 2); break;
        case '\n': append_str("\\n", 2);  break;
        case '\r': append_str("\\r", 2);  break;
        case '\t': append_str("\\t", 2);  break;
        default:
            if (c < 0x20) {
                char esc[8];
                auto n = static_cast<size_t>(::snprintf(esc, sizeof(esc), "\\u%04x", c));
                append_str(esc, n);
            } else {
                dest.push_back(static_cast<char>(c));
            }
            break;
        }
    }
}

} // namespace aivision
