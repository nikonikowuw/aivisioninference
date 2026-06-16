#include "monitor/device_monitor.h"
#include "monitor/device_snapshot_json.h"
#include <nlohmann/json.hpp>
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

using json = nlohmann::json;

int main()
{
    std::cout << "=== Test: DeviceSnapshot JSON Serialization ===" << std::endl;

    aivision::monitor::DeviceSnapshot snapshot;
    snapshot.timestamp_ms = 123456789;
    snapshot.info.hostname = "test-host";
    snapshot.info.cpu_model = "Intel Core i7";
    snapshot.info.cpu_cores = 8;
    snapshot.info.gpu_model = "NVIDIA RTX 3080";
    snapshot.info.total_memory = 16ULL * 1024 * 1024 * 1024;

    snapshot.metrics.cpu_usage.available = true;
    snapshot.metrics.cpu_usage.value = 25.5;
    snapshot.metrics.cpu_usage.unit = "percent";
    snapshot.metrics.cpu_usage.source = "/proc/stat";

    snapshot.metrics.memory_usage.available = false;
    snapshot.metrics.memory_usage.error = "Failed to read meminfo";

    aivision::monitor::AcceleratorInfo acc;
    acc.type = "gpu";
    acc.vendor = "nvidia";
    acc.name = "NVIDIA GeForce RTX 3080";
    acc.usage.available = true;
    acc.usage.value = 45.0;
    acc.usage.unit = "percent";
    acc.usage.source = "nvidia-smi";
    snapshot.accelerators.push_back(acc);

    aivision::monitor::ProbeDiagnostic diag;
    diag.name = "linux_generic";
    diag.success = true;
    diag.evidence = {"Evidence 1"};
    snapshot.diagnostics.push_back(diag);

    // Test ToHeartbeatJson
    std::string hb_str = aivision::monitor::ToHeartbeatJson(3600, 2, "1.0.0", "rkmpp", "[]", snapshot);
    try {
        json hb = json::parse(hb_str);
        TEST("hb has uptime", hb["uptime"] == 3600);
        TEST("hb has cpu_usage", hb["cpu_usage"] == 25.5);
        TEST("hb has memory_usage as 0.0 when unavailable", hb["memory_usage"] == 0.0);
        TEST("hb has legacy hardware_info cpu_model", hb["hardware_info"]["cpu_model"] == "Intel Core i7");
        TEST("hb has device_info hostname", hb["device_info"]["hostname"] == "test-host");
        TEST("hb has device_metrics cpu_usage value", hb["device_metrics"]["cpu_usage"]["value"] == 25.5);
        TEST("hb has device_metrics memory_usage error", hb["device_metrics"]["memory_usage"]["error"] == "Failed to read meminfo");
        TEST("hb has accelerators list", hb["accelerators"].is_array() && hb["accelerators"].size() == 1);
        TEST("hb has probe_diagnostics active_probes", hb["probe_diagnostics"]["active_probes"][0] == "linux_generic");
    } catch (const std::exception& e) {
        TEST("hb json parsing failed", false);
    }

    // Test ToEngineDeviceApiJson
    std::string api_str = aivision::monitor::ToEngineDeviceApiJson(3600, 2, "1.0.0", "rkmpp", snapshot);
    try {
        json api = json::parse(api_str);
        TEST("api has runtime section", api.contains("runtime"));
        TEST("api has device section", api.contains("device"));
        TEST("api runtime status is ok", api["runtime"]["status"] == "ok");
        TEST("api device timestamp_ms is correct", api["device"]["timestamp_ms"] == 123456789);
        TEST("api device diagnostics", api["device"]["diagnostics"][0]["name"] == "linux_generic");
    } catch (const std::exception& e) {
        TEST("api json parsing failed", false);
    }

    // Test ToLegacyHardwareInfoJson
    std::string legacy_str = aivision::monitor::ToLegacyHardwareInfoJson("rkmpp", snapshot);
    try {
        json legacy = json::parse(legacy_str);
        TEST("legacy has hal_platform", legacy["hal_platform"] == "rkmpp");
        TEST("legacy has cpu_model", legacy["cpu_model"] == "Intel Core i7");
        TEST("legacy has gpu_model", legacy["gpu_model"] == "NVIDIA RTX 3080");
    } catch (const std::exception& e) {
        TEST("legacy json parsing failed", false);
    }

    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
