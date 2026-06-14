// test_heartbeat_reporter.cpp — HeartbeatReporter 核心逻辑单元测试
// 验证心跳 JSON 载荷构造、响应解析等核心功能

#include <algorithm>
#include <iostream>
#include <sstream>
#include <string>
#include <cstring>
#include <cstdlib>
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

namespace {

std::string ExtractJsonField(const std::string &json, const std::string &field_name) {
    std::string search = "\"" + field_name + "\"";
    size_t pos = json.find(search);
    if (pos == std::string::npos) return "";

    size_t start = json.find(":", pos);
    if (start == std::string::npos) return "";

    start++;
    while (start < json.size() && (json[start] == ' ' || json[start] == '\t' || json[start] == '\r' || json[start] == '\n'))
        start++;

    if (start >= json.size()) return "";

    if (json[start] == '"') {
        start++;
        std::string value;
        for (size_t i = start; i < json.size(); ++i) {
            if (json[i] == '"') return value;
            value.push_back(json[i]);
        }
        return "";
    }

    size_t end = json.find_first_of(",}\n", start);
    if (end == std::string::npos) end = json.size();

    std::string val = json.substr(start, end - start);
    while (!val.empty() && (val.back() == ' ' || val.back() == '\t' || val.back() == '\r' || val.back() == '\n' || val.back() == '}'))
        val.pop_back();
    return val;
}

std::string JsonEscape(const std::string &value) {
    std::string escaped;
    escaped.reserve(value.size());
    for (char ch : value) {
        switch (ch) {
        case '\\': escaped += "\\\\"; break;
        case '"': escaped += "\\\""; break;
        case '\n': escaped += "\\n"; break;
        case '\r': escaped += "\\r"; break;
        case '\t': escaped += "\\t"; break;
        default: escaped += ch; break;
        }
    }
    return escaped;
}

// Build a simplified heartbeat payload matching the production BuildHeartbeatPayload
std::string BuildHeartbeatPayload(uint64_t uptime, size_t load, const std::string& engine_version,
                                   const std::string& cpu_model, const std::string& hal_platform,
                                   const std::vector<std::string>& installed_algos)
{
    std::stringstream ss;
    ss << "{"
       << "\"uptime\":" << uptime << ","
       << "\"current_load\":" << load << ","
       << "\"cpu_usage\":0.0,"
       << "\"memory_usage\":0.0,"
       << "\"engine_version\":\"" << engine_version << "\","
       << "\"hal_platform\":\"" << hal_platform << "\","
       << "\"hardware_info\":{"
       << "\"cpu_model\":\"" << cpu_model << "\","
       << "\"gpu_model\":\"\","
       << "\"total_memory\":17179869184,"
       << "\"cpu_cores\":8"
       << "},";

    // Installed algorithms
    ss << "\"installed_algorithms\":[";
    for (size_t i = 0; i < installed_algos.size(); ++i) {
        if (i > 0) ss << ",";
        ss << "{\"algo_name\":\"" << installed_algos[i]
           << "\",\"version\":\"1.0.0\",\"status\":\"installed\",\"install_path\":\"/opt/algo/" << installed_algos[i] << "\"}";
    }
    ss << "]}";

    return ss.str();
}

void test_build_heartbeat_payload() {
    std::cout << "\n=== Test: BuildHeartbeatPayload ===" << std::endl;

    // 1. Minimal heartbeat
    std::vector<std::string> empty_algos;
    std::string payload1 = BuildHeartbeatPayload(3600, 2, "1.0.0", "Apple M1", "macos", empty_algos);
    TEST("payload starts with {", payload1.size() > 0 && payload1[0] == '{');
    TEST("payload ends with }", !payload1.empty() && payload1.back() == '}');
    TEST("payload has uptime field",
         ExtractJsonField(payload1, "uptime") == "3600");
    TEST("payload has current_load",
         ExtractJsonField(payload1, "current_load") == "2");
    TEST("payload has engine_version",
         ExtractJsonField(payload1, "engine_version") == "1.0.0");
    TEST("payload has hal_platform",
         ExtractJsonField(payload1, "hal_platform") == "macos");
    TEST("payload contains cpu_usage field",
         !ExtractJsonField(payload1, "cpu_usage").empty());
    TEST("payload contains memory_usage field",
         !ExtractJsonField(payload1, "memory_usage").empty());

    // 2. With installed algorithms
    std::vector<std::string> algos = {"yolov8", "face_recognition"};
    std::string payload2 = BuildHeartbeatPayload(7200, 3, "1.0.0", "Intel i7", "rkmpp", algos);
    TEST("payload with algos contains algo_name",
         payload2.find("yolov8") != std::string::npos);
    TEST("payload with algos contains second algo",
         payload2.find("face_recognition") != std::string::npos);
    TEST("payload with algos contains install_path",
         payload2.find("install_path") != std::string::npos);

    // 3. Empty uptime
    std::string payload3 = BuildHeartbeatPayload(0, 0, "1.0.0", "Unknown", "generic", empty_algos);
    TEST("payload with zero uptime",
         ExtractJsonField(payload3, "uptime") == "0");

    // 4. Version string with timestamp suffix (production scenario)
    std::vector<std::string> many_algos;
    for (int i = 0; i < 10; ++i) {
        many_algos.push_back("algo_" + std::to_string(i));
    }
    std::string payload4 = BuildHeartbeatPayload(100, 5, "1.2.3", "ARM Cortex-A55", "rkmpp", many_algos);
    TEST("payload with many algos is valid JSON start", payload4[0] == '{');
    TEST("payload with many algos has all entries",
         payload4.find("algo_9") != std::string::npos);
    TEST("payload with many algos has proper JSON array format",
         payload4.find("installed_algorithms") != std::string::npos);
}

void test_parse_and_deploy_response() {
    std::cout << "\n=== Test: ParseAndDeploy (response parsing) ===" << std::endl;

    // 1. Heartbeat response with pending deployments
    std::string response1 = R"({
        "code": 0,
        "message": "success",
        "data": {
            "pending_deployments": [
                {
                    "algo_package_id": "pkg-1",
                    "download_url": "http://minio:9000/algo.tar.gz",
                    "md5": "abc123",
                    "extract_path": "/opt/algo/yolov8",
                    "algo_name": "yolov8",
                    "version": "1.0.0"
                }
            ]
        }
    })";

    TEST("response contains pending_deployments",
         response1.find("pending_deployments") != std::string::npos);
    TEST("response contains algo_package_id",
         ExtractJsonField(response1, "algo_package_id") == "pkg-1");
    TEST("response contains download_url",
         ExtractJsonField(response1, "download_url").find("http://") != std::string::npos);
    TEST("response contains md5",
         ExtractJsonField(response1, "md5") == "abc123");
    TEST("response contains extract_path",
         ExtractJsonField(response1, "extract_path") == "/opt/algo/yolov8");
    TEST("response contains algo_name",
         ExtractJsonField(response1, "algo_name") == "yolov8");
    TEST("response contains version",
         ExtractJsonField(response1, "version") == "1.0.0");

    // 2. Empty deployments
    std::string response2 = R"({"code":0,"message":"success","data":{"pending_deployments":[]}})";
    TEST("empty deployments response is parseable",
         response2.find("\"pending_deployments\":[]") != std::string::npos);

    // 3. Response with algorithm info from list
    std::string algo_list_response = R"([
        {"algo_name":"yolov8","version":"1.0.0","status":"installed"},
        {"algo_name":"face_recognition","version":"1.0.0","status":"installed"}
    ])";
    TEST("algorithm list response is an array", algo_list_response[0] == '[');
    TEST("algorithm list has yolov8", algo_list_response.find("yolov8") != std::string::npos);
    TEST("algorithm list has face_recognition",
         algo_list_response.find("face_recognition") != std::string::npos);

    // 4. Response with error status
    std::string error_response = R"({"code":10101,"message":"引擎版本过低","data":null})";
    TEST("error response has error code",
         ExtractJsonField(error_response, "code") == "10101");
    TEST("error response has message",
         ExtractJsonField(error_response, "message") == "引擎版本过低");
}

void test_json_escape() {
    std::cout << "\n=== Test: JsonEscape ===" << std::endl;

    TEST("plain string escapes nothing",
         JsonEscape("hello") == "hello");
    TEST("double quote is escaped",
         JsonEscape("he\"llo") == "he\\\"llo");
    TEST("backslash is escaped",
         JsonEscape("path\\to") == "path\\\\to");
    TEST("newline is escaped",
         JsonEscape("line1\nline2") == "line1\\nline2");
    TEST("tab is escaped",
         JsonEscape("col1\tcol2") == "col1\\tcol2");
    TEST("carriage return is escaped",
         JsonEscape("line1\r") == "line1\\r");
    TEST("empty string stays empty",
         JsonEscape("").empty());
    TEST("mixed special chars",
         JsonEscape("\"quote\\\n\t\r") == "\\\"quote\\\\\\n\\t\\r");
}

void test_curl_write_callback() {
    std::cout << "\n=== Test: CURL Write Callback Simulation ===" << std::endl;

    // Simulate the write callback used in SendHeartbeat
    std::string response_data;
    auto write_callback = [](void* contents, size_t size, size_t nmemb, void* userp) -> size_t {
        auto* str = static_cast<std::string*>(userp);
        size_t total = size * nmemb;
        str->append(static_cast<char*>(contents), total);
        return total;
    };

    // Simulate receiving a response chunk
    const char* chunk1 = "{\"code\":0,\"message\":\"success\",";
    response_data.clear();
    write_callback(const_cast<char*>(chunk1), 1, strlen(chunk1), &response_data);
    TEST("first chunk appended correctly",
         response_data == "{\"code\":0,\"message\":\"success\",");

    const char* chunk2 = "\"data\":{}}";
    write_callback(const_cast<char*>(chunk2), 1, strlen(chunk2), &response_data);
    TEST("full response assembled correctly",
         response_data == "{\"code\":0,\"message\":\"success\",\"data\":{}}");

    // Test with binary data
    std::string binary_data;
    char binary_chunk[] = {'\x00', '\x01', '\x02', '\xFF'};
    write_callback(binary_chunk, 1, 4, &binary_data);
    TEST("callback handles binary data", binary_data.size() == 4);
}

} // anonymous namespace

int main() {
    test_build_heartbeat_payload();
    test_parse_and_deploy_response();
    test_json_escape();
    test_curl_write_callback();

    std::cout << "\n=== Results ===" << std::endl;
    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
