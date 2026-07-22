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
    std::string val = GetEnvString(name);
    if (val.empty()) return default_value;
    for (auto &c : val) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    return (val == "1" || val == "true" || val == "yes" || val == "on");
}

static uint32_t GetEnvUInt32(const char *name, uint32_t default_value)
{
    std::string val = GetEnvString(name);
    if (val.empty()) return default_value;
    try {
        return static_cast<uint32_t>(std::stoul(val));
    } catch (...) {
        return default_value;
    }
}

static uint32_t GetPositiveEnvUInt32(const char *name)
{
    std::string value = GetEnvString(name);
    if (value.empty()) {
        std::cerr << "[Config] " << name << " is missing; preview scheduling disabled" << std::endl;
        return 0;
    }
    try {
        size_t parsed = 0;
        unsigned long result = std::stoul(value, &parsed);
        if (parsed != value.size() || result == 0 || result > UINT32_MAX)
            throw std::out_of_range("not a positive uint32");
        return static_cast<uint32_t>(result);
    } catch (...) {
        std::cerr << "[Config] " << name << " must be a positive integer; preview scheduling disabled" << std::endl;
        return 0;
    }
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
    config.worker_count = GetEnvUInt32("NIKO_ENGINE_WORKERS", config.worker_count);
    config.enable_ffmpeg_fallback = GetEnvBool("NIKO_ENGINE_ENABLE_FFMPEG_FALLBACK", config.enable_ffmpeg_fallback);
    config.metrics_interval_ms = GetEnvUInt32("NIKO_ENGINE_METRICS_MS", config.metrics_interval_ms);
	config.max_preview_streams = GetPositiveEnvUInt32("NIKO_ENGINE_MAX_PREVIEW_STREAMS");
    config.rtsp_push_server = GetEnvString("NIKO_ENGINE_RTSP_PUSH", config.rtsp_push_server);
    config.zlm_api_url = GetEnvString("NIKO_ENGINE_ZLM_URL", config.zlm_api_url);
    config.zlm_secret = GetEnvString("NIKO_ENGINE_ZLM_SECRET", config.zlm_secret);

    config.platform_url = GetEnvString("NIKO_ENGINE_PLATFORM_URL", config.platform_url);
    config.node_id = GetEnvString("NIKO_ENGINE_NODE_ID", config.node_id);
    config.auth_token = GetEnvString("NIKO_ENGINE_AUTH_TOKEN", config.auth_token);
    config.algo_dir = GetEnvString("NIKO_ENGINE_ALGO_DIR", config.algo_dir);
    config.enable_mqtt = GetEnvBool("NIKO_ENGINE_ENABLE_MQTT", config.enable_mqtt);
    config.mqtt_broker = GetEnvString("NIKO_ENGINE_MQTT_BROKER", config.mqtt_broker);
    config.mqtt_client_id = GetEnvString("NIKO_ENGINE_MQTT_CLIENT_ID", config.mqtt_client_id);
    config.mqtt_username = GetEnvString("NIKO_ENGINE_MQTT_USER", config.mqtt_username);
    config.mqtt_password = GetEnvString("NIKO_ENGINE_MQTT_PASS", config.mqtt_password);

    config.device_platform = GetEnvString("NIKO_ENGINE_DEVICE_PLATFORM", config.device_platform);
    std::string default_storage = config.algo_dir;
    if (default_storage.empty()) {
        default_storage = "/";
    }
    config.device_storage_path = GetEnvString("NIKO_ENGINE_DEVICE_STORAGE_PATH", default_storage);
    config.device_enable_external_commands = GetEnvBool("NIKO_ENGINE_DEVICE_ENABLE_COMMANDS", config.device_enable_external_commands);
    config.device_command_timeout_ms = GetEnvUInt32("NIKO_ENGINE_DEVICE_COMMAND_TIMEOUT_MS", config.device_command_timeout_ms);

    return config;
}

static std::string DisplayVal(const std::string& val, const std::string& empty_label = "<unset>") {
    if (val.empty()) {
        return empty_label;
    }
    return val;
}

static std::string DisplayBool(bool val, const std::string& t_label = "true", const std::string& f_label = "false") {
    if (val) {
        return t_label;
    }
    return f_label;
}

static void PrintRuntimeConfig(const EngineConfig &config,
                               const std::string &env_file)
{
    std::string env_display = env_file;
    if (env_display.empty()) {
        env_display = GetEnvString("NIKO_ENGINE_ENV_FILE", ".env");
    }
    std::cout << "[Config] env_file=" << env_display << std::endl;
    std::cout << "[Config] workers=" << config.worker_count
              << " metrics_ms=" << config.metrics_interval_ms
			  << " max_preview_streams=" << config.max_preview_streams << std::endl;
    std::cout << "[Config] ffmpeg_fallback=" << DisplayBool(config.enable_ffmpeg_fallback, "enabled", "disabled")
              << " rtsp_push=" << config.rtsp_push_server
              << " zlm_url=" << config.zlm_api_url
              << " zlm_secret=" << DisplayVal(config.zlm_secret, "<empty>") << std::endl;
    std::cout << "[Config] platform_url=" << DisplayVal(config.platform_url)
              << " node_id=" << DisplayVal(config.node_id)
              << " auth_token=" << DisplayVal(config.auth_token, "<empty>") << std::endl;
    std::cout << "[Config] algo_dir=" << config.algo_dir << std::endl;
    std::cout << "[Config] enable_mqtt=" << DisplayBool(config.enable_mqtt)
              << " mqtt_broker=" << config.mqtt_broker
              << " mqtt_client_id=" << config.mqtt_client_id
              << " mqtt_user=" << config.mqtt_username
              << " mqtt_pass=" << DisplayVal(config.mqtt_password, "<empty>") << std::endl;
    std::cout << "[Config] device_platform=" << DisplayVal(config.device_platform)
              << " device_storage_path=" << config.device_storage_path
              << " device_enable_commands=" << DisplayBool(config.device_enable_external_commands)
              << " device_command_timeout_ms=" << config.device_command_timeout_ms << std::endl;
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
    std::cout << "  --env-file PATH      Load environment variables from file (default: .env)" << std::endl;
    std::cout << "  --workers N          Worker thread count (default: 4)" << std::endl;
    std::cout << "  --disable-ffmpeg-fallback  Disable playback-only FFmpeg relay fallback (enabled by default)" << std::endl;
    std::cout << "  --metrics-ms N       Metrics report interval in ms (default: 5000)" << std::endl;
    std::cout << "  --rtsp-push URL      RTSP publish base URL (default: rtsp://localhost:10554)" << std::endl;
    std::cout << "  --zlm-url URL        ZLM API URL (default: http://localhost:80)" << std::endl;
    std::cout << "  --zlm-secret SECRET  ZLM API secret" << std::endl;
    std::cout << "  --enable-mqtt        Enable native MQTT client" << std::endl;
    std::cout << "  --mqtt-broker URL    MQTT broker URL (default: tcp://localhost:1883)" << std::endl;
    std::cout << "  --mqtt-client-id ID  MQTT client ID" << std::endl;
    std::cout << "  --mqtt-user USER     MQTT username" << std::endl;
    std::cout << "  --mqtt-pass PASS     MQTT password" << std::endl;
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
        else if (arg == "--workers" && i + 1 < argc)
        {
            config.worker_count = static_cast<uint32_t>(std::stoul(argv[++i]));
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
        else if (arg == "--enable-mqtt")
        {
            config.enable_mqtt = true;
        }
        else if (arg == "--mqtt-broker" && i + 1 < argc)
        {
            config.mqtt_broker = argv[++i];
        }
        else if (arg == "--mqtt-client-id" && i + 1 < argc)
        {
            config.mqtt_client_id = argv[++i];
        }
        else if (arg == "--mqtt-user" && i + 1 < argc)
        {
            config.mqtt_username = argv[++i];
        }
        else if (arg == "--mqtt-pass" && i + 1 < argc)
        {
            config.mqtt_password = argv[++i];
        }
        else
        {
            std::cerr << "Unknown option: " << arg << std::endl;
            PrintUsage(argv[0]);
            return 1;
        }
    }

    PrintVersion();
    PrintRuntimeConfig(config, env_file);

    // 缓存 ZLM URL 解析结果，避免运行时重复解析
    config.zlm_url_info = ParseZLMUrl(config.zlm_api_url);

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

    std::cout << "Engine initialized, starting main loop..." << std::endl;

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
