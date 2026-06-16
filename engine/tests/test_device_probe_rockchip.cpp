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
    std::cout << "=== Test: RockchipProbe ===" << std::endl;

    auto probe = aivision::monitor::CreateRockchipProbe();
    TEST("Probe name is rockchip", probe->Name() == "rockchip");

    aivision::monitor::DeviceMonitorConfig config;
    config.proc_dir = "fixtures/device_probe";
    config.sys_dir = "fixtures/device_probe";

#ifdef __APPLE__
    auto det = probe->Detect(config);
    TEST("Detect returns unsupported on macOS", !det.supported);
#else
    // On Linux/Rockchip platforms
    auto det = probe->Detect(config);
    // Should be unsupported on a standard Linux x86 developer machine
    TEST("Detect returns unsupported on generic Linux", !det.supported || det.confidence < 70);
#endif

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
