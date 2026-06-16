#include "probes/device_probe.h"
#include <iostream>
#include <vector>

static int passed = 0;
static int failed = 0;

#define TEST(name, expr) do { \
    if (!(expr)) { \
        std::cerr << "[FAIL] " << name << std::endl; \
        failed++; \
    } else { \
        std::cout << "[PASS] " << name << std::endl; \
        passed++; \
    } \
} while(0)

int main()
{
    std::cout << "=== Test: MacOSProbe ===" << std::endl;

    auto probe = aivision::monitor::CreateMacOSProbe();
    TEST("Probe name is macos", probe->Name() == "macos");

    aivision::monitor::DeviceMonitorConfig config;
    config.storage_path = "/";

#ifdef __APPLE__
    auto det = probe->Detect(config);
    TEST("Detect returns supported on macOS", det.supported);

    aivision::monitor::DeviceStaticInfo info;
    probe->CollectStaticInfo(config, info);
    TEST("Static info hostname is not empty", !info.hostname.empty());
    TEST("Static info cpu_cores is greater than 0", info.cpu_cores > 0);
    TEST("Static info total_memory is greater than 0", info.total_memory > 0);
    TEST("Static info gpu_model is Apple GPU", info.gpu_model == "Apple GPU");
    TEST("Static info npu_model is Apple Neural Engine", info.npu_model == "Apple Neural Engine");

    aivision::monitor::DeviceDynamicMetrics metrics;
    std::vector<aivision::monitor::AcceleratorInfo> accelerators;
    
    // First sample
    probe->CollectDynamicMetrics(config, metrics, accelerators);
    TEST("First sample cpu_usage is unavailable (waiting for second sample)", !metrics.cpu_usage.available);
    TEST("Memory usage is available", metrics.memory_usage.available);
    TEST("Memory usage is between 0 and 100", metrics.memory_usage.value >= 0.0 && metrics.memory_usage.value <= 100.0);
    TEST("Storage usage is available", metrics.storage_usage.available);
    TEST("Storage usage is between 0 and 100", metrics.storage_usage.value >= 0.0 && metrics.storage_usage.value <= 100.0);

    // Sleep a bit and run second sample
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    probe->CollectDynamicMetrics(config, metrics, accelerators);
    TEST("Second sample cpu_usage is available", metrics.cpu_usage.available);
    TEST("CPU usage is between 0 and 100", metrics.cpu_usage.value >= 0.0 && metrics.cpu_usage.value <= 100.0);

    TEST("Accelerators list has at least Apple GPU and Apple NPU", accelerators.size() >= 2);
    bool found_gpu = false;
    bool found_npu = false;
    for (const auto& acc : accelerators) {
        if (acc.vendor == "apple" && acc.type == "gpu") found_gpu = true;
        if (acc.vendor == "apple" && acc.type == "npu") found_npu = true;
    }
    TEST("Found Apple GPU in accelerators", found_gpu);
    TEST("Found Apple NPU in accelerators", found_npu);
#else
    auto det = probe->Detect(config);
    TEST("Detect returns unsupported on non-macOS", !det.supported);
#endif

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
