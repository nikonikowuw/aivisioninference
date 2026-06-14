// aivision-engine — C++ 推理数据面引擎入口。
// 控制面 (Go) 与数据面 (C++) 分离架构的数据面进程。

#include <algorithm>
#include <cctype>
#include <csignal>
#include <cstdlib>
#include <fstream>
#include <iostream>

#include "engine.h"

using namespace aivision;

namespace
{

// 信号标志 (async-signal-safe)
static volatile sig_atomic_t g_signal_received = 0;

static std::string Trim(const std::string &value)
{
    size_t begin = value.find_first_not_of(" \t\r\n");
    if (begin == std::string::npos)
        return "";
    size_t end = value.find_last_not_of(" \t\r\n");
    return value.substr(begin, end - begin + 1);
}

static std::string GetEnvString(const char *name, const std::string &default_value = "")
{
    const char *value = std::getenv(name);
    if (!value || value[0] == '\0')
        return default_value;
    return value;
}

static std::string StripEnvValue(std::string value)
{
    value = Trim(value);
    if (value.size() >= 2)
    {
        char quote = value.front();
        if ((quote == '\'' || quote == '"') && value.back() == quote)
            return value.substr(1, value.size() - 2);
    }
    size_t comment = value.find('#');
    if (comment != std::string::npos)
        value = value.substr(0, comment);
    return Trim(value);
}

static void LoadDotEnvFile(const std::string &path)
{
    std::ifstream file(path);
    if (!file.good())
        return;

    std::string line;
    while (std::getline(file, line))
    {
        line = Trim(line);
        if (line.empty() || line[0] == '#')
            continue;
        if (line.rfind("export ", 0) == 0)
            line = Trim(line.substr(7));

        size_t eq = line.find('=');
        if (eq == std::string::npos || eq == 0)
            continue;

        std::string key = Trim(line.substr(0, eq));
        std::string value = StripEnvValue(line.substr(eq + 1));
        if (key.empty() || std::getenv(key.c_str()))
            continue;
        setenv(key.c_str(), value.c_str(), 0);
    }
}

static void LoadDotEnv()
{
    std::string path = GetEnvString("NIKO_ENGINE_ENV_FILE", ".env");
    LoadDotEnvFile(path);
}

static bool GetEnvBool(const char *name, bool default_value)
{
    std::string value = GetEnvString(name);
    if (value.empty())
        return default_value;
    std::transform(value.begin(), value.end(), value.begin(), [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
    if (value == "1" || value == "true" || value == "yes" || value == "on")
        return true;
    if (value == "0" || value == "false" || value == "no" || value == "off")
        return false;
    return default_value;
}

static uint32_t GetEnvUInt32(const char *name, uint32_t default_value)
{
    std::string value = GetEnvString(name);
    if (value.empty())
        return default_value;
    try
    {
        return static_cast<uint32_t>(std::stoul(value));
    }
    catch (...)
    {
        std::cerr << "Invalid uint32 env " << name << "=" << value << ", using default " << default_value << std::endl;
        return default_value;
    }
}

static std::string JoinPath(const std::string &dir, const std::string &file)
{
    if (dir.empty())
        return file;
    if (dir.back() == '/')
        return dir + file;
    return dir + "/" + file;
}

static std::string NormalizePlatform(std::string platform)
{
    std::transform(platform.begin(), platform.end(), platform.begin(), [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
    platform.erase(std::remove_if(platform.begin(), platform.end(), [](char c) { return c == '-' || c == '_' || c == ' '; }), platform.end());
    return platform;
}

static std::string DefaultHalPathForPlatform(const std::string &platform)
{
    const std::string normalized = NormalizePlatform(platform);
    const std::string hal_dir = GetEnvString("NIKO_ENGINE_HAL_DIR", "/usr/local/lib/aivision");
    if (normalized.empty() || normalized == "none")
        return "";
    if (normalized == "mac" || normalized == "macos" || normalized == "apple" || normalized == "applesilicon" || normalized == "mseries" || normalized == "videotoolbox")
        return JoinPath(hal_dir, "libaivision-hal-macos-videotoolbox.dylib");
    if (normalized == "rk" || normalized == "rknn" || normalized == "rkmpp" || normalized == "rockchip" || normalized == "rk3568" || normalized == "rk3588")
        return JoinPath(hal_dir, "libaivision-hal-rkmpp.so");
    if (normalized == "ascend" || normalized == "atlas" || normalized == "huawei" || normalized == "cann")
        return JoinPath(hal_dir, "libaivision-hal-ascend.so");
    return platform;
}

static std::string FindEnvFileArg(int argc, char *argv[])
{
    for (int i = 1; i < argc; ++i)
    {
        std::string arg = argv[i];
        if (arg == "--env-file" && i + 1 < argc)
            return argv[i + 1];
    }
    return "";
}

static EngineConfig LoadConfigFromEnv()
{
    EngineConfig config;
    config.ipc_addr = GetEnvString("NIKO_ENGINE_IPC_ADDR", GetEnvString("NIKO_ENGINE_ADDR", config.ipc_addr));
    config.worker_count = GetEnvUInt32("NIKO_ENGINE_WORKERS", config.worker_count);
    config.hal_so_path = GetEnvString("NIKO_ENGINE_HAL_SO", config.hal_so_path);
    config.fallback_hal_so_path = GetEnvString("NIKO_ENGINE_FALLBACK_HAL_SO", config.fallback_hal_so_path);
    config.hal_config_json = GetEnvString("NIKO_ENGINE_HAL_CONFIG", config.hal_config_json);
    config.enable_ffmpeg_fallback = GetEnvBool("NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK", config.enable_ffmpeg_fallback);
    config.metrics_interval_ms = GetEnvUInt32("NIKO_ENGINE_METRICS_MS", config.metrics_interval_ms);
    config.rtsp_push_server = GetEnvString("NIKO_ENGINE_RTSP_PUSH", config.rtsp_push_server);
    config.zlm_api_url = GetEnvString("NIKO_ENGINE_ZLM_URL", config.zlm_api_url);
    config.zlm_secret = GetEnvString("NIKO_ENGINE_ZLM_SECRET", config.zlm_secret);

    config.platform_url = GetEnvString("NIKO_ENGINE_PLATFORM_URL", config.platform_url);
    config.node_id = GetEnvString("NIKO_ENGINE_NODE_ID", config.node_id);
    config.auth_token = GetEnvString("NIKO_ENGINE_AUTH_TOKEN", config.auth_token);
    config.http_port = GetEnvUInt32("NIKO_ENGINE_HTTP_PORT", config.http_port);
    config.algo_dir = GetEnvString("NIKO_ENGINE_ALGO_DIR", config.algo_dir);

    std::string hal_platform = GetEnvString("NIKO_ENGINE_HAL_PLATFORM");
    if (config.hal_so_path.empty() && !hal_platform.empty())
        config.hal_so_path = DefaultHalPathForPlatform(hal_platform);

    std::string fallback_platform = GetEnvString("NIKO_ENGINE_FALLBACK_HAL_PLATFORM");
    if (config.fallback_hal_so_path.empty() && !fallback_platform.empty())
        config.fallback_hal_so_path = DefaultHalPathForPlatform(fallback_platform);

    return config;
}

static void PrintRuntimeConfig(const EngineConfig &config,
                               const std::string &env_file,
                               const std::string &hal_platform,
                               const std::string &fallback_hal_platform)
{
    std::string env_display = env_file.empty() ? GetEnvString("NIKO_ENGINE_ENV_FILE", ".env") : env_file;
    std::cout << "[Config] env_file=" << env_display << std::endl;
    std::cout << "[Config] ipc_addr=" << config.ipc_addr
              << " workers=" << config.worker_count
              << " metrics_ms=" << config.metrics_interval_ms << std::endl;
    std::cout << "[Config] hal_platform=" << (hal_platform.empty() ? "<unset>" : hal_platform)
              << " hal_so=" << (config.hal_so_path.empty() ? "<unset>" : config.hal_so_path) << std::endl;
    std::cout << "[Config] fallback_hal_platform=" << (fallback_hal_platform.empty() ? "<unset>" : fallback_hal_platform)
              << " fallback_hal_so=" << (config.fallback_hal_so_path.empty() ? "<unset>" : config.fallback_hal_so_path) << std::endl;
    std::cout << "[Config] ffmpeg_fallback=" << (config.enable_ffmpeg_fallback ? "enabled" : "disabled")
              << " rtsp_push=" << config.rtsp_push_server
              << " zlm_url=" << config.zlm_api_url
              << " zlm_secret=" << (config.zlm_secret.empty() ? "<empty>" : "<set>") << std::endl;
    std::cout << "[Config] platform_url=" << (config.platform_url.empty() ? "<unset>" : config.platform_url)
              << " node_id=" << (config.node_id.empty() ? "<unset>" : config.node_id)
              << " auth_token=" << (config.auth_token.empty() ? "<unset>" : "<set>") << std::endl;
    std::cout << "[Config] http_port=" << config.http_port << std::endl;
    std::cout << "[Config] algo_dir=" << config.algo_dir << std::endl;
}

} // namespace

// 信号处理 (仅设置标志，不调用非 async-signal-safe 函数)
static void SignalHandler(int sig)
{
    g_signal_received = sig;
}

// 打印版本信息
static void PrintVersion()
{
    std::cout << "aivision-engine v" << InferenceEngine::Version() << std::endl;
    std::cout << "C++ Inference Data Plane Engine" << std::endl;
}

// 打印使用帮助
static void PrintUsage(const char *prog)
{
    std::cout << "Usage: " << prog << " [options]" << std::endl;
    std::cout << "Options:" << std::endl;
    std::cout << "  --addr HOST:PORT     IPC listen address (default: 0.0.0.0:9500)" << std::endl;
    std::cout << "  --env-file PATH      Load environment variables from file (default: .env)" << std::endl;
    std::cout << "  --workers N          Worker thread count (default: 4)" << std::endl;
    std::cout << "  --hal-platform NAME  HAL platform name: macos/rkmpp/ascend" << std::endl;
    std::cout << "  --hal-so PATH        HAL platform pipeline .so/.dylib path" << std::endl;
    std::cout << "  --fallback-hal-platform NAME  Fallback HAL platform name" << std::endl;
    std::cout << "  --fallback-hal-so PATH  Fallback HAL .so/.dylib path; must decode and output frames for inference" << std::endl;
    std::cout << "  --hal-config JSON    HAL configuration JSON" << std::endl;
    std::cout << "  --disable-ffmpeg-fallback  Disable playback-only FFmpeg relay fallback (enabled by default)" << std::endl;
    std::cout << "  --metrics-ms N       Metrics report interval in ms (default: 5000)" << std::endl;
    std::cout << "  --rtsp-push URL      RTSP publish base URL (default: rtsp://localhost:10554)" << std::endl;
    std::cout << "  --zlm-url URL        ZLM API URL (default: http://localhost:80)" << std::endl;
    std::cout << "  --zlm-secret SECRET  ZLM API secret" << std::endl;
    std::cout << "  --version            Print version and exit" << std::endl;
    std::cout << "  --help               Print this help and exit" << std::endl;
}

int main(int argc, char *argv[])
{
    std::string env_file = FindEnvFileArg(argc, argv);
    if (!env_file.empty())
        LoadDotEnvFile(env_file);
    else
        LoadDotEnv();

    // 配置加载优先级：默认值 < .env < 环境变量 < 命令行参数
    EngineConfig config = LoadConfigFromEnv();
    std::string hal_platform = GetEnvString("NIKO_ENGINE_HAL_PLATFORM");
    std::string fallback_hal_platform = GetEnvString("NIKO_ENGINE_FALLBACK_HAL_PLATFORM");

    for (int i = 1; i < argc; ++i)
    {
        std::string arg = argv[i];
        if (arg == "--help")
        {
            PrintUsage(argv[0]);
            return 0;
        }
        else if (arg == "--version")
        {
            PrintVersion();
            return 0;
        }
        else if (arg == "--env-file" && i + 1 < argc)
        {
            ++i;
        }
        else if (arg == "--addr" && i + 1 < argc)
        {
            config.ipc_addr = argv[++i];
        }
        else if (arg == "--workers" && i + 1 < argc)
        {
            config.worker_count = static_cast<uint32_t>(std::stoul(argv[++i]));
        }
        else if (arg == "--hal-platform" && i + 1 < argc)
        {
            hal_platform = argv[++i];
            config.hal_so_path = DefaultHalPathForPlatform(hal_platform);
        }
        else if (arg == "--hal-so" && i + 1 < argc)
        {
            config.hal_so_path = argv[++i];
        }
        else if (arg == "--fallback-hal-platform" && i + 1 < argc)
        {
            fallback_hal_platform = argv[++i];
            config.fallback_hal_so_path = DefaultHalPathForPlatform(fallback_hal_platform);
        }
        else if (arg == "--fallback-hal-so" && i + 1 < argc)
        {
            config.fallback_hal_so_path = argv[++i];
        }
        else if (arg == "--hal-config" && i + 1 < argc)
        {
            config.hal_config_json = argv[++i];
        }
        else if (arg == "--disable-ffmpeg-fallback")
        {
            config.enable_ffmpeg_fallback = false;
        }
        else if (arg == "--metrics-ms" && i + 1 < argc)
        {
            config.metrics_interval_ms = static_cast<uint32_t>(std::stoul(argv[++i]));
        }
        else if (arg == "--rtsp-push" && i + 1 < argc)
        {
            config.rtsp_push_server = argv[++i];
        }
        else if (arg == "--zlm-url" && i + 1 < argc)
        {
            config.zlm_api_url = argv[++i];
        }
        else if (arg == "--zlm-secret" && i + 1 < argc)
        {
            config.zlm_secret = argv[++i];
        }
        else
        {
            std::cerr << "Unknown option: " << arg << std::endl;
            PrintUsage(argv[0]);
            return 1;
        }
    }

    PrintVersion();
    PrintRuntimeConfig(config, env_file, hal_platform, fallback_hal_platform);

    // 注册信号处理
    std::signal(SIGINT, SignalHandler);
    std::signal(SIGTERM, SignalHandler);

    // 创建并初始化引擎
    InferenceEngine engine(config);

    if (!engine.Initialize())
    {
        std::cerr << "Engine initialization failed" << std::endl;
        return 1;
    }

    std::cout << "Engine initialized, starting IPC server on "
              << config.ipc_addr << "..." << std::endl;

    // 运行主循环 (阻塞)
    engine.Run([]() { return g_signal_received != 0; });

    // 检查信号标志 (async-signal-safe)
    if (g_signal_received)
    {
        std::cerr << "Received signal " << g_signal_received << ", shutting down..." << std::endl;
    }

    engine.Shutdown();

    std::cout << "Engine shut down gracefully" << std::endl;
    return 0;
}
