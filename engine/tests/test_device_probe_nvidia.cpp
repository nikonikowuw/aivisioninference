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
    std::cout << "=== Test: NvidiaProbe ===" << std::endl;

    auto probe = aivision::monitor::CreateNvidiaProbe();
    TEST("Probe name is nvidia", probe->Name() == "nvidia");

    aivision::monitor::DeviceMonitorConfig config;
    config.proc_dir = "fixtures/device_probe";
    config.sys_dir = "fixtures/device_probe";

#ifdef __APPLE__
    auto det = probe->Detect(config);
    TEST("Detect returns unsupported on macOS", !det.supported);
#else
    // On Linux
    auto det = probe->Detect(config);
    // On development machine without GPU, should be unsupported
    std::cout << "Nvidia supported on current system: " << det.supported << std::endl;
#endif

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
