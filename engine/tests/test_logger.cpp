// Logger quick smoke test — verifies console + file output.
#include "logger/logger.h"
#include "logger/json_formatter.h"

#include <cstdio>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <string>

using namespace aivision;

int main()
{
    // Phase 1: console
    Logger::InitConsole(LogLevel::Debug);
    LOG_INFO("[Test] Console only — should appear on terminal");

    // Phase 2: file sinks
    Logger::InitFileSinks("/tmp/aivision-log-test", LogLevel::Trace, LogLevel::Error, LogLevel::Info);
    LOG_INFO("[Test] File+console — this goes to app.log too");
    LOG_WARN("[Test] Warning message");
    LOG_ERROR("[Test] Error message — should go to both app.log and error.log");

    Logger::Shutdown();

    // Verify files
    auto check_file = [](const std::string &path, const std::string &label) -> bool {
        std::ifstream f(path);
        if (!f.good()) {
            std::cerr << "FAIL: " << label << " not found at " << path << std::endl;
            return false;
        }
        std::string content((std::istreambuf_iterator<char>(f)),
                             std::istreambuf_iterator<char>());
        if (content.empty()) {
            std::cerr << "FAIL: " << label << " is empty" << std::endl;
            return false;
        }
        std::cout << "PASS: " << label << " (" << content.size() << " bytes)" << std::endl;
        std::cout << "  First line: " << content.substr(0, content.find('\n')) << std::endl;
        return true;
    };

    // Find today's date dir
    time_t now = time(nullptr);
    struct tm tm_buf;
    gmtime_r(&now, &tm_buf);
    char date_dir[32];
    snprintf(date_dir, sizeof(date_dir), "%04d-%02d/%02d",
             1900 + tm_buf.tm_year, tm_buf.tm_mon + 1, tm_buf.tm_mday);

    std::string base = "/tmp/aivision-log-test/";
    std::string app_log = base + date_dir + "/app.log";
    std::string err_log = base + date_dir + "/error.log";

    bool ok = true;
    ok &= check_file(app_log, "app.log");
    ok &= check_file(err_log, "error.log");

    // Cleanup
    std::remove(app_log.c_str());
    std::remove(err_log.c_str());

    return ok ? 0 : 1;
}
