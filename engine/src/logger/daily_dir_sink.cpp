// 每日目录 sink 实现。
// 按 UTC 日期在 logs/{YYYY-MM}/{DD}/ 下创建文件，日期变化时自动切换。

#include "logger/daily_dir_sink.h"

#include <ctime>
#include <filesystem>
#include <iomanip>
#include <sstream>
#include <unistd.h>

namespace aivision {

template<typename Mutex>
DailyDirSinkMT<Mutex>::DailyDirSinkMT(const std::string &base_path,
                                       const std::string &filename,
                                       spdlog::level::level_enum min_level)
    : spdlog::sinks::base_sink<Mutex>()
    , base_path_(base_path)
    , filename_(filename)
{
    this->set_level(min_level);
}

template<typename Mutex>
DailyDirSinkMT<Mutex>::~DailyDirSinkMT()
{
    if (file_stream_.is_open()) {
        file_stream_.flush();
        file_stream_.close();
    }
}

template<typename Mutex>
void DailyDirSinkMT<Mutex>::sink_it_(const spdlog::details::log_msg &msg)
{
    // 从消息时间戳中提取 UTC 日期
    auto msg_time = std::chrono::system_clock::to_time_t(msg.time);
    struct tm tm_buf = {};
    ::gmtime_r(&msg_time, &tm_buf);

    int year  = tm_buf.tm_year + 1900;
    int month = tm_buf.tm_mon + 1;
    int day   = tm_buf.tm_mday;

    // 日期变化时创建新目录并打开新文件
    if (year != current_year_ || month != current_month_ || day != current_day_) {
        if (file_stream_.is_open()) {
            file_stream_.flush();
            file_stream_.close();
        }

        std::ostringstream dir_path;
        dir_path << base_path_
                 << "/" << std::setw(4) << std::setfill('0') << year
                 << "-" << std::setw(2) << std::setfill('0') << month
                 << "/" << std::setw(2) << std::setfill('0') << day;

        EnsureDirectories(dir_path.str());

        std::string full_path = dir_path.str() + "/" + filename_;
        file_stream_.open(full_path, std::ios::app);

        current_year_  = year;
        current_month_ = month;
        current_day_   = day;
    }

    if (!file_stream_.is_open()) {
        return; // 静默跳过
    }

    // 应用 formatter（JsonFormatter），写入格式化后的内容
    spdlog::memory_buf_t formatted;
    if (this->formatter_) {
        this->formatter_->format(msg, formatted);
        file_stream_.write(formatted.data(),
                           static_cast<std::streamsize>(formatted.size()));
    }
}

template<typename Mutex>
void DailyDirSinkMT<Mutex>::flush_()
{
    if (file_stream_.is_open()) {
        file_stream_.flush();
    }
}

template<typename Mutex>
void DailyDirSinkMT<Mutex>::EnsureDirectories(const std::string &path)
{
    std::error_code ec;
    std::filesystem::create_directories(path, ec);
    if (ec) {
        static const char prefix[] = "[Logger] Failed to create log directory: ";
        (void)::write(STDERR_FILENO, prefix, sizeof(prefix) - 1);
        (void)::write(STDERR_FILENO, ec.message().c_str(), ec.message().size());
        (void)::write(STDERR_FILENO, "\n", 1);
    }
}

// 显式实例化
template class DailyDirSinkMT<std::mutex>;

} // namespace aivision
