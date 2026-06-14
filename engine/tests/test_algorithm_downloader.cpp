// test_algorithm_downloader.cpp — AlgorithmDownloader 核心逻辑单元测试
// 测试 MD5 校验、tar.gz 解压、DownloadFile 回调等

#include <algorithm>
#include <cassert>
#include <cstring>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>
#include <cstdlib>
#include <cstdio>

#include <openssl/md5.h>
#include <unistd.h>

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

std::string ComputeMD5(const std::string& file_path) {
    std::ifstream file(file_path, std::ios::binary);
    if (!file.is_open())
        return "";

    MD5_CTX md5Context;
    MD5_Init(&md5Context);

    char buffer[1024 * 16];
    while (file.good()) {
        file.read(buffer, sizeof(buffer));
        MD5_Update(&md5Context, buffer, file.gcount());
    }

    unsigned char result[MD5_DIGEST_LENGTH];
    MD5_Final(result, &md5Context);

    std::stringstream ss;
    for (int i = 0; i < MD5_DIGEST_LENGTH; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(result[i]);
    }
    return ss.str();
}

bool CalculateAndCompareMD5(const std::string& file_path, const std::string& expected_md5) {
    std::string calculated = ComputeMD5(file_path);
    std::string expected = expected_md5;
    std::transform(expected.begin(), expected.end(), expected.begin(), ::tolower);
    return calculated == expected;
}

std::string CreateTempFile(const std::string& content) {
    char tmp_name[] = "/tmp/aivision_test_XXXXXX";
    int fd = mkstemp(tmp_name);
    if (fd == -1) return "";
    write(fd, content.data(), content.size());
    close(fd);
    return std::string(tmp_name);
}

bool ExtractTarGz(const std::string& tar_path, const std::string& dest_dir) {
    std::string cmd = "tar -xzf " + tar_path + " -C " + dest_dir;
    int status = std::system(cmd.c_str());
    return status == 0;
}

} // anonymous namespace

void test_md5_computation() {
    std::cout << "\n=== Test: MD5 Computation ===" << std::endl;

    // 1. Empty file
    std::string empty_path = CreateTempFile("");
    TEST("empty file MD5 is d41d8cd98f00b204e9800998ecf8427e",
         ComputeMD5(empty_path) == "d41d8cd98f00b204e9800998ecf8427e");
    std::remove(empty_path.c_str());

    // 2. Known content
    std::string hello_path = CreateTempFile("Hello, World!");
    std::string md5 = ComputeMD5(hello_path);
    // MD5("Hello, World!") = 65a8e27d8879283831b664bd8b7f0ad4
    TEST("Hello World MD5 is 65a8e27d8879283831b664bd8b7f0ad4",
         md5 == "65a8e27d8879283831b664bd8b7f0ad4");
    std::remove(hello_path.c_str());

    // 3. Verify MD5 matches expected
    std::string test_path = CreateTempFile("test content for md5 verification");
    std::string test_md5 = ComputeMD5(test_path);
    TEST("MD5 comparison succeeds with same hash",
         CalculateAndCompareMD5(test_path, test_md5));
    TEST("MD5 comparison fails with different hash",
         !CalculateAndCompareMD5(test_path, "00000000000000000000000000000000"));
    std::remove(test_path.c_str());

    // 4. Binary content
    std::vector<char> binary(256);
    for (int i = 0; i < 256; ++i) binary[i] = static_cast<char>(i);
    std::string bin_path = CreateTempFile(std::string(binary.data(), binary.size()));
    std::string bin_md5 = ComputeMD5(bin_path);
    TEST("binary file produces non-empty MD5", !bin_md5.empty());
    TEST("binary MD5 is 32 hex chars", bin_md5.length() == 32);
    std::remove(bin_path.c_str());

    // 5. Nonexistent file returns empty
    TEST("nonexistent file returns empty MD5",
         ComputeMD5("/tmp/nonexistent_file_xyz").empty());
}

void test_tar_extraction() {
    std::cout << "\n=== Test: tar.gz Extraction ===" << std::endl;

    // Create a temp directory for extraction
    std::string extract_dir = "/tmp/aivision_test_extract";
    std::string cmd = "rm -rf " + extract_dir + " && mkdir -p " + extract_dir;
    std::system(cmd.c_str());

    // Create a test file and tar.gz archive
    std::string test_file = CreateTempFile("algorithm library content");
    std::string tar_path = "/tmp/aivision_test_archive.tar.gz";
    std::string create_tar = "tar -czf " + tar_path + " -C /tmp " + test_file.substr(5);
    int status = std::system(create_tar.c_str());
    TEST("tar archive created", status == 0);

    // Extract the archive
    bool extracted = ExtractTarGz(tar_path, extract_dir);
    TEST("tar.gz extraction succeeds", extracted);

    // Verify the extracted file exists
    std::string extracted_path = extract_dir + "/" + test_file.substr(5);
    std::ifstream extracted_file(extracted_path);
    TEST("extracted file exists", extracted_file.good());

    if (extracted_file.good()) {
        std::string content((std::istreambuf_iterator<char>(extracted_file)),
                             std::istreambuf_iterator<char>());
        TEST("extracted content matches", content == "algorithm library content");
    }

    // Cleanup
    std::remove(tar_path.c_str());
    std::remove(test_file.c_str());
    std::system(("rm -rf " + extract_dir).c_str());
}

void test_download_behavior() {
    std::cout << "\n=== Test: Download Behavior (callback simulation) ===" << std::endl;

    // Simulate the write callback used in DownloadFile
    std::string downloaded_content;
    auto write_callback = [](void* contents, size_t size, size_t nmemb, void* userp) -> size_t {
        auto* buffer = static_cast<std::string*>(userp);
        buffer->append(static_cast<char*>(contents), size * nmemb);
        return size * nmemb;
    };

    // Test write callback with sample data
    const char* test_data = "HTTP response chunk";
    downloaded_content.clear();
    size_t result = write_callback(const_cast<char*>(test_data), 1, strlen(test_data), &downloaded_content);
    TEST("callback returns correct size", result == strlen(test_data));
    TEST("callback appends content to string", downloaded_content == "HTTP response chunk");

    // Test multiple chunks
    downloaded_content.clear();
    write_callback(const_cast<char*>("chunk1"), 1, 6, &downloaded_content);
    write_callback(const_cast<char*>("chunk2"), 1, 6, &downloaded_content);
    TEST("callback appends multiple chunks", downloaded_content == "chunk1chunk2");
}

int main() {
    test_md5_computation();
    test_tar_extraction();
    test_download_behavior();

    std::cout << "\n=== Results ===" << std::endl;
    std::cout << "Passed: " << passed << ", Failed: " << failed << std::endl;
    return failed > 0 ? 1 : 0;
}
