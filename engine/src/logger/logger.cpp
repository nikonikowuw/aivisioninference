// Logger 全局单例实现。
//
// 两阶段初始化:
//   Phase 1 — InitConsole(): 尽早调用，仅终端输出。
//   Phase 2 — InitFileSinks(): 配置加载后调用，追加文件 sink 并下调终端级别。

#include "logger/logger.h"
#include "logger/daily_dir_sink.h"
#include "logger/json_formatter.h"

namespace aivision {

// 静态成员定义
std::shared_ptr<spdlog::logger> Logger::instance_;
std::shared_ptr<spdlog::sinks::stdout_color_sink_mt> Logger::console_sink_;
std::once_flag Logger::init_console_flag_;

void Logger::InitConsole(LogLevel level)
{
    std::call_once(init_console_flag_, [level]() {
        // 创建终端 sink
        console_sink_ = std::make_shared<spdlog::sinks::stdout_color_sink_mt>();
        console_sink_->set_level(spdlog::level::trace); // 由 Logger 级别控制

        // 设置 glog 风格 pattern: I20260722 14:30:01.123456 12345 main.cpp:42]
        // %L=级别首字母, %Y%m%d=日期, %H:%M:%S.%e=时间+微秒, %t=线程ID, %s:%#=文件:行号
        console_sink_->set_pattern("%L%Y%m%d %H:%M:%S.%e %t %s:%#] %v");

        // 创建 logger 实例，只有一个 terminal sink
        instance_ = std::make_shared<spdlog::logger>("aivision-engine",
                                                     spdlog::sinks_init_list{console_sink_});
        instance_->set_level(static_cast<spdlog::level::level_enum>(level));
        instance_->flush_on(spdlog::level::err); // ERROR 及以上立即 flush
    });
}

void Logger::InitFileSinks(const std::string &log_dir,
                           LogLevel app_level,
                           LogLevel error_level,
                           LogLevel console_level)
{
    if (!instance_) {
        InitConsole(LogLevel::Debug); // 安全兜底
    }

    // --- 创建 app.log sink (所有级别) ---
    auto app_sink = std::make_shared<DailyDirSink>(log_dir, "app.log");
    app_sink->set_level(static_cast<spdlog::level::level_enum>(app_level));
    app_sink->set_formatter(std::make_unique<JsonFormatter>());

    // --- 创建 error.log sink (仅 ERROR+FATAL) ---
    auto error_sink = std::make_shared<DailyDirSink>(log_dir, "error.log");
    error_sink->set_level(static_cast<spdlog::level::level_enum>(error_level));
    error_sink->set_formatter(std::make_unique<JsonFormatter>());

    // --- 追加到 logger ---
    instance_->sinks().push_back(std::move(app_sink));
    instance_->sinks().push_back(std::move(error_sink));

    // --- 调整终端级别为生产级别 ---
    SetConsoleLevel(console_level);
}

std::shared_ptr<spdlog::logger> Logger::Instance()
{
    // 如果未初始化，兜底自动初始化
    if (!instance_) {
        InitConsole(LogLevel::Info);
    }
    return instance_;
}

void Logger::SetConsoleLevel(LogLevel level)
{
    if (console_sink_) {
        console_sink_->set_level(static_cast<spdlog::level::level_enum>(level));
    }
}

void Logger::Shutdown()
{
    if (instance_) {
        instance_->flush();
    }
    spdlog::shutdown();
    instance_.reset();
    console_sink_.reset();

    // 重置 once_flag 以便重新初始化
    // 注意: std::once_flag 没有 reset() 方法，但 Shutdown 在进程退出时调用，
    // 通常不需要重新初始化。若需要重新初始化，可考虑改用简单的静态布尔标志。
    // 这里保持 once_flag 不复位 —— 进程退出场景足够。
}

} // namespace aivision
