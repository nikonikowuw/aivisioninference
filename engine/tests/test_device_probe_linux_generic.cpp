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
    std::cout << "=== Test: LinuxGenericProbe ===" << std::endl;

    auto probe = aivision::monitor::CreateLinuxGenericProbe();
    TEST("Probe name is linux_generic", probe->Name() == "linux_generic");

    aivision::monitor::DeviceMonitorConfig config1;
    config1.proc_dir = "fixtures/device_probe/linux_sample1";
    config1.storage_path = "/";

    aivision::monitor::DeviceMonitorConfig config2;
    config2.proc_dir = "fixtures/device_probe/linux_sample2";
    config2.storage_path = "/";

#ifdef __APPLE__
    // On macOS, LinuxGenericProbe should report unsupported
    auto det = probe->Detect(config1);
    TEST("Detect returns unsupported on macOS", !det.supported);
    
    aivision::monitor::DeviceStaticInfo info;
    probe->CollectStaticInfo(config1, info);
    TEST("Static info cpu_cores remains default on macOS", info.cpu_cores == 0);

    aivision::monitor::DeviceDynamicMetrics metrics;
    std::vector<aivision::monitor::AcceleratorInfo> accelerators;
    probe->CollectDynamicMetrics(config1, metrics, accelerators);
    TEST("Dynamic metrics cpu_usage remains unavailable on macOS", !metrics.cpu_usage.available);
#else
    // On Linux, we expect full support and parsing
    auto det = probe->Detect(config1);
    TEST("Detect returns supported on Linux", det.supported);

    aivision::monitor::DeviceStaticInfo info;
    probe->CollectStaticInfo(config1, info);
    TEST("Static info cpu_cores is 2", info.cpu_cores == 2);
    TEST("Static info cpu_model matches fixture", info.cpu_model == "Intel(R) Core(TM) i7-12700H");
    TEST("Static info total_memory is 16384000000 bytes", info.total_memory == 16000000ULL * 1024);

    aivision::monitor::DeviceDynamicMetrics metrics;
    std::vector<aivision::monitor::AcceleratorInfo> accelerators;
    
    // First sample
    probe->CollectDynamicMetrics(config1, metrics, accelerators);
    TEST("First sample cpu_usage is unavailable (waiting for second sample)", !metrics.cpu_usage.available);
    TEST("Memory usage available", metrics.memory_usage.available);
    TEST("Memory usage value is 50.0%", metrics.memory_usage.value == 50.0);
    TEST("Storage usage available", metrics.storage_usage.available);

    // Second sample
    probe->CollectDynamicMetrics(config2, metrics, accelerators);
    TEST("Second sample cpu_usage is available", metrics.cpu_usage.available);
    TEST("CPU usage value is exactly 24.0%", metrics.cpu_usage.value > 23.9 && metrics.cpu_usage.value < 24.1);
#endif

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
