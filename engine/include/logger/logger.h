// aivision-engine 日志系统入口。
// 全局单例 + LOG_* 宏，替换 engine 中所有 std::cout / std::cerr。
//
// 两阶段初始化:
//   Phase 1 — InitConsole(): 尽早调用，仅终端输出，默认 DEBUG 级别。
//   Phase 2 — InitFileSinks(): 配置加载后调用，追加 JSON 文件 sink，下调终端级别。

#ifndef AIVISION_LOGGER_H
#define AIVISION_LOGGER_H

#include <memory>
#include <mutex>
#include <string>

#include <spdlog/spdlog.h>
#include <spdlog/sinks/stdout_color_sinks.h>

namespace aivision {

/// 引擎日志级别（与 spdlog::level 对齐）
enum class LogLevel {
    Trace = 0,
    Debug = 1,
    Info  = 2,
    Warn  = 3,
    Error = 4,
    Fatal = 5
};

/// 日志系统全局单例。
class Logger {
public:
    /// Phase 1: 初始化终端输出（尽早调用）。
    /// 输出格式: I20260722 14:30:01.123456 12345 main.cpp:42] ...
    static void InitConsole(LogLevel level = LogLevel::Debug);

    /// Phase 2: 初始化 JSON 文件输出（配置加载后调用）。
    /// 追加文件 sink，同时将终端级别调整到生产级别。
    /// @param log_dir       日志根目录（如 "logs" 或 "/var/log/aivision"）
    /// @param app_level     app.log 最低级别
    /// @param error_level   error.log 最低级别
    /// @param console_level 终端级别
    static void InitFileSinks(
        const std::string &log_dir,
        LogLevel app_level   = LogLevel::Trace,
        LogLevel error_level = LogLevel::Error,
        LogLevel console_level = LogLevel::Info
    );

    /// 获取全局 logger 实例。
    static std::shared_ptr<spdlog::logger> Instance();

    /// 运行时调整终端日志级别。
    static void SetConsoleLevel(LogLevel level);

    /// Shutdown: 刷新并关闭所有 sink。
    static void Shutdown();

private:
    static std::shared_ptr<spdlog::logger> instance_;
    static std::shared_ptr<spdlog::sinks::stdout_color_sink_mt> console_sink_;
    static std::once_flag init_console_flag_;
};

// Convenience macros — 替换所有 std::cout / std::cerr。
// 使用示例:
//   LOG_INFO("[MQTT] Connected to broker {}", url);
//   LOG_ERROR("[Worker] Infer failed for algo: {}", algo_name);
#define LOG_TRACE(...)  SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::trace, __VA_ARGS__)
#define LOG_DEBUG(...)  SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::debug, __VA_ARGS__)
#define LOG_INFO(...)   SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::info, __VA_ARGS__)
#define LOG_WARN(...)   SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::warn, __VA_ARGS__)
#define LOG_ERROR(...)  SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::err, __VA_ARGS__)
#define LOG_FATAL(...)  SPDLOG_LOGGER_CALL(::aivision::Logger::Instance().get(), spdlog::level::critical, __VA_ARGS__)

} // namespace aivision

#endif // AIVISION_LOGGER_H
