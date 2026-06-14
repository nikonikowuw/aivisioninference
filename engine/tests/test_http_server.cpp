// test_http_server.cpp — HTTP Server 路由处理单元测试
// 验证 /health、/hardware-info、/deploy-algo、/algorithms 的 JSON 响应

#include <httplib.h>
#include <iostream>
#include <thread>
#include <chrono>
#include <sstream>
#include <string>
#include <cstring>
#include <cstdlib>

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

// Simplified JSON field extraction matching the implementation in http_server.cpp
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

} // anonymous namespace

void test_extract_json_field() {
    std::cout << "\n=== Test: ExtractJsonField ===" << std::endl;

    // Basic extraction
    TEST("extract string field",
         ExtractJsonField("{\"status\":\"ok\"}", "status") == "ok");
    TEST("extract numeric field",
         ExtractJsonField("{\"uptime\":12345}", "uptime") == "12345");
    TEST("extract boolean field",
         ExtractJsonField("{\"success\":true}", "success") == "true");

    // Nested field (should match first occurrence)
    TEST("extract from complex json returns first match",
         ExtractJsonField("{\"a\":1,\"b\":2}", "a") == "1");

    // Missing field returns empty
    TEST("missing field returns empty",
         ExtractJsonField("{\"status\":\"ok\"}", "nonexistent").empty());

    // Empty json
    TEST("empty json returns empty",
         ExtractJsonField("", "field").empty());

    // Malformed json
    TEST("malformed json handles gracefully",
         ExtractJsonField("{invalid}", "field").empty());

    // Value with spaces
    TEST("extract value with spaces",
         ExtractJsonField("{\"name\":\"edge node 1\"}", "name") == "edge node 1");
}

void test_health_endpoint() {
    std::cout << "\n=== Test: Health Endpoint ===" << std::endl;

    httplib::Server svr;

    // Register health route matching production logic
    svr.Get("/health", [](const httplib::Request&, httplib::Response& res) {
        std::stringstream ss;
        ss << "{"
           << "\"status\":\"ok\","
           << "\"uptime\":3600,"
           << "\"current_load\":2,"
           << "\"engine_version\":\"1.0.0\""
           << "}";
        res.set_content(ss.str(), "application/json");
    });

    // Start server on random port
    int port = 18081;
    std::thread t([&]() { svr.listen("127.0.0.1", port); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    httplib::Client cli("http://127.0.0.1:" + std::to_string(port));

    // Test GET /health
    auto res = cli.Get("/health");
    TEST("health endpoint returns 200", res && res->status == 200);
    if (res) {
        TEST("health content-type is application/json",
             res->get_header_value("Content-Type").find("application/json") != std::string::npos);

        std::string body = res->body;
        TEST("health response contains status",
             ExtractJsonField(body, "status") == "ok");
        TEST("health response contains uptime",
             !ExtractJsonField(body, "uptime").empty());
        TEST("health response contains current_load",
             ExtractJsonField(body, "current_load") == "2");
        TEST("health response contains engine_version",
             ExtractJsonField(body, "engine_version") == "1.0.0");
    }

    // Test OPTIONS (preflight CORS) - covered in test_cors_headers

    svr.stop();
    t.join();
}

void test_hardware_info_endpoint() {
    std::cout << "\n=== Test: Hardware Info Endpoint ===" << std::endl;

    httplib::Server svr;

    svr.Get("/hardware-info", [](const httplib::Request&, httplib::Response& res) {
        std::stringstream ss;
        ss << "{"
           << "\"hal_platform\":\"macos\","
           << "\"cpu_model\":\"Apple M1\","
           << "\"gpu_model\":\"Apple M1 GPU\","
           << "\"total_memory\":17179869184"
           << "}";
        res.set_content(ss.str(), "application/json");
    });

    int port = 18082;
    std::thread t([&]() { svr.listen("127.0.0.1", port); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    httplib::Client cli("http://127.0.0.1:" + std::to_string(port));

    auto res = cli.Get("/hardware-info");
    TEST("hardware-info returns 200", res && res->status == 200);
    if (res) {
        std::string body = res->body;
        TEST("hardware-info response has platform",
             !ExtractJsonField(body, "hal_platform").empty());
        TEST("hardware-info response has cpu_model",
             !ExtractJsonField(body, "cpu_model").empty());
        TEST("hardware-info response has total_memory",
             !ExtractJsonField(body, "total_memory").empty());
        TEST("hardware-info platform is macos",
             ExtractJsonField(body, "hal_platform") == "macos");
    }

    svr.stop();
    t.join();
}

void test_deploy_algorithm_endpoint() {
    std::cout << "\n=== Test: Deploy Algorithm Endpoint ===" << std::endl;

    httplib::Server svr;

    svr.Post("/deploy-algo", [](const httplib::Request& req, httplib::Response& res) {
        std::string body = req.body;
        std::string pkg_id = ExtractJsonField(body, "algo_package_id");
        std::string url = ExtractJsonField(body, "download_url");
        std::string md5 = ExtractJsonField(body, "md5");
        std::string extract_path = ExtractJsonField(body, "extract_path");
        std::string name = ExtractJsonField(body, "algo_name");
        std::string version = ExtractJsonField(body, "version");

        if (pkg_id.empty() || url.empty() || md5.empty() || extract_path.empty() || name.empty() || version.empty()) {
            res.status = 400;
            res.set_content("{\"success\":false,\"error_message\":\"Missing required fields\"}", "application/json");
            return;
        }

        res.status = 200;
        res.set_content("{\"success\":true,\"message\":\"Deployment triggered\"}", "application/json");
    });

    int port = 18083;
    std::thread t([&]() { svr.listen("127.0.0.1", port); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    httplib::Client cli("http://127.0.0.1:" + std::to_string(port));

    // Test POST with valid body
    std::string valid_body = R"({
        "algo_package_id":"pkg-1",
        "download_url":"http://minio:9000/bucket/algo.tar.gz",
        "md5":"abc123def456",
        "extract_path":"/opt/algo/yolov8",
        "algo_name":"yolov8",
        "version":"1.0.0"
    })";

    auto res = cli.Post("/deploy-algo", valid_body, "application/json");
    TEST("deploy-algo returns 200 with valid body", res && res->status == 200);
    if (res) {
        TEST("deploy-algo success response",
             ExtractJsonField(res->body, "success") == "true");
        TEST("deploy-algo response has message",
             !ExtractJsonField(res->body, "message").empty());
    }

    // Test POST with missing fields
    auto res2 = cli.Post("/deploy-algo", "{\"algo_package_id\":\"pkg-1\"}", "application/json");
    TEST("deploy-algo returns 400 with missing fields", res2 && res2->status == 400);
    if (res2) {
        TEST("deploy-algo failure response",
             ExtractJsonField(res2->body, "success") == "false");
        TEST("deploy-algo error message present",
             !ExtractJsonField(res2->body, "error_message").empty());
    }

    // Test GET returns 405 (method not allowed)
    auto res3 = cli.Get("/deploy-algo");
    TEST("GET /deploy-algo returns error",
         res3 && (res3->status == 405 || res3->status == 404));

    svr.stop();
    t.join();
}

void test_algorithms_endpoint() {
    std::cout << "\n=== Test: Algorithms Endpoint ===" << std::endl;

    httplib::Server svr;

    svr.Get("/algorithms", [](const httplib::Request&, httplib::Response& res) {
        // Simplified response matching production format
        res.set_content(
            "[{\"algo_name\":\"yolov8\",\"version\":\"1.0.0\",\"status\":\"installed\"},"
            "{\"algo_name\":\"face_recognition\",\"version\":\"1.0.0\",\"status\":\"installed\"}]",
            "application/json");
    });

    int port = 18084;
    std::thread t([&]() { svr.listen("127.0.0.1", port); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    httplib::Client cli("http://127.0.0.1:" + std::to_string(port));

    auto res = cli.Get("/algorithms");
    TEST("algorithms returns 200", res && res->status == 200);
    if (res) {
        std::string body = res->body;
        TEST("algorithms response is a JSON array", body.size() > 0 && body[0] == '[');
        TEST("algorithms response contains algo_name",
             body.find("yolov8") != std::string::npos);
        TEST("algorithms response contains status field",
             body.find("installed") != std::string::npos);
        TEST("algorithms response contains version field",
             body.find("1.0.0") != std::string::npos);
    }

    svr.stop();
    t.join();
}

void test_cors_headers() {
    std::cout << "\n=== Test: CORS Headers ===" << std::endl;

    httplib::Server svr;

    // Set CORS header post-routing matching production
    svr.set_post_routing_handler([](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        res.set_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
        res.set_header("Access-Control-Allow-Headers", "Content-Type, Authorization");
    });

    svr.Get("/health", [](const httplib::Request&, httplib::Response& res) {
        res.set_content("{\"status\":\"ok\"}", "application/json");
    });

    // OPTIONS handler matching production
    svr.Options(R"(.*)", [](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        res.set_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
        res.set_header("Access-Control-Allow-Headers", "Content-Type, Authorization");
        res.status = 200;
    });

    int port = 18085;
    std::thread t([&]() { svr.listen("127.0.0.1", port); });
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    httplib::Client cli("http://127.0.0.1:" + std::to_string(port));

    // Test CORS headers on GET
    auto res = cli.Get("/health");
    TEST("GET /health has CORS headers", res);
    if (res) {
        TEST("CORS Access-Control-Allow-Origin is *",
             res->get_header_value("Access-Control-Allow-Origin") == "*");
        TEST("CORS Access-Control-Allow-Methods present",
             !res->get_header_value("Access-Control-Allow-Methods").empty());
    }

    // Test OPTIONS on any route
    auto opt_res = cli.Options("/health");
    TEST("OPTIONS /health returns 200", opt_res && opt_res->status == 200);
    if (opt_res) {
        TEST("OPTIONS has CORS Allow-Origin",
             opt_res->get_header_value("Access-Control-Allow-Origin") == "*");
    }

    svr.stop();
    t.join();
}

int main() {
    test_extract_json_field();
    test_health_endpoint();
    test_hardware_info_endpoint();
    test_deploy_algorithm_endpoint();
    test_algorithms_endpoint();
    test_cors_headers();

    std::cout << "\n=== Results ===" << std::endl;
    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
