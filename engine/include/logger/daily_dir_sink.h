// 按日期目录拆分的文件 sink。
//
// 输出路径: {base_path}/{YYYY-MM}/{DD}/{filename}
// 日志消息到达时检查 UTC 日期，日期变化时自动创建新目录并打开新文件。
// 线程安全（继承 base_sink<Mutex>）。

#ifndef AIVISION_DAILY_DIR_SINK_H
#define AIVISION_DAILY_DIR_SINK_H

#include <chrono>
#include <ctime>
#include <fstream>
#include <mutex>
#include <string>

#include <spdlog/sinks/base_sink.h>
#include <spdlog/details/null_mutex.h>

namespace aivision {

/// 每日目录文件 sink。
template<typename Mutex>
class DailyDirSinkMT : public spdlog::sinks::base_sink<Mutex> {
public:
    DailyDirSinkMT(const std::string &base_path,
                   const std::string &filename,
                   spdlog::level::level_enum min_level = spdlog::level::trace);
    ~DailyDirSinkMT() override;

protected:
    void sink_it_(const spdlog::details::log_msg &msg) override;
    void flush_() override;

private:
    /// 创建 YYYY-MM/DD 嵌套目录。
    static void EnsureDirectories(const std::string &path);

    std::string base_path_;
    std::string filename_;
    std::ofstream file_stream_;
    int current_year_  = 0;
    int current_month_ = 0;
    int current_day_   = 0;
};

using DailyDirSink = DailyDirSinkMT<std::mutex>;

} // namespace aivision

#endif // AIVISION_DAILY_DIR_SINK_H
