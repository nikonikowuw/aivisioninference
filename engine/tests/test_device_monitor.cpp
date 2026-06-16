#include "monitor/device_monitor.h"
#include <iostream>

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
    std::cout << "=== Test: DeviceMonitor ===" << std::endl;

    aivision::monitor::DeviceMonitorConfig config;
    config.platform_override = "linux_generic";
    config.storage_path = "/";
    config.enable_external_commands = false;

    aivision::monitor::DeviceMonitor monitor(config);
    
    // Test Initialize
    bool init_ret = monitor.Initialize();
    // Since we forced "linux_generic" platform_override, it should succeed
    TEST("DeviceMonitor initialize", init_ret == true);

    // Test GetSnapshot
    auto snapshot = monitor.GetSnapshot();
    TEST("Snapshot timestamp exists", snapshot.timestamp_ms > 0);
    TEST("Default metrics cpu_usage unavailable at start", !snapshot.metrics.cpu_usage.available);

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
