// aivision-engine — C++ 推理数据面引擎入口。
// 控制面 (Go) 与数据面 (C++) 分离架构的数据面进程。

#include <iostream>
#include <csignal>
#include <cstdlib>
#include <chrono>
#include <thread>

#include "engine.h"

using namespace aivision;

// 全局引擎指针 (用于信号处理)
static InferenceEngine *g_engine = nullptr;

// 信号标志 (async-signal-safe)
static volatile sig_atomic_t g_signal_received = 0;

// 信号处理 (仅设置标志，不调用非 async-signal-safe 函数)
static void SignalHandler(int sig)
{
    g_signal_received = sig;
    if (g_engine)
    {
        g_engine->RequestShutdown();
    }
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
    std::cout << "  --workers N          Worker thread count (default: 4)" << std::endl;
    std::cout << "  --hal-so PATH        HAL platform pipeline .so path" << std::endl;
    std::cout << "  --hal-config JSON    HAL configuration JSON" << std::endl;
    std::cout << "  --metrics-ms N       Metrics report interval in ms (default: 5000)" << std::endl;
    std::cout << "  --zlm-url URL        ZLM API URL (default: http://localhost:80)" << std::endl;
    std::cout << "  --zlm-secret SECRET  ZLM API secret" << std::endl;
    std::cout << "  --version            Print version and exit" << std::endl;
    std::cout << "  --help               Print this help and exit" << std::endl;
}

int main(int argc, char *argv[])
{
    // 解析命令行参数
    EngineConfig config;

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
        else if (arg == "--addr" && i + 1 < argc)
        {
            config.ipc_addr = argv[++i];
        }
        else if (arg == "--workers" && i + 1 < argc)
        {
            config.worker_count = static_cast<uint32_t>(std::stoul(argv[++i]));
        }
        else if (arg == "--hal-so" && i + 1 < argc)
        {
            config.hal_so_path = argv[++i];
        }
        else if (arg == "--hal-config" && i + 1 < argc)
        {
            config.hal_config_json = argv[++i];
        }
        else if (arg == "--metrics-ms" && i + 1 < argc)
        {
            config.metrics_interval_ms = static_cast<uint32_t>(std::stoul(argv[++i]));
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

    // 注册信号处理
    std::signal(SIGINT, SignalHandler);
    std::signal(SIGTERM, SignalHandler);

    // 创建并初始化引擎
    InferenceEngine engine(config);
    g_engine = &engine;

    if (!engine.Initialize())
    {
        std::cerr << "Engine initialization failed" << std::endl;
        return 1;
    }

    std::cout << "Engine initialized, starting IPC server on "
              << config.ipc_addr << "..." << std::endl;

    // 运行主循环 (阻塞)
    engine.Run();

    // 检查信号标志 (async-signal-safe)
    if (g_signal_received)
    {
        std::cerr << "Received signal " << g_signal_received << ", shutting down..." << std::endl;
    }

    std::cout << "Engine shut down gracefully" << std::endl;
    g_engine = nullptr;
    return 0;
}
